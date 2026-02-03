package logic

import (
	"context"
	"fmt"
	"strings"
	"zero-service/common/dockerx"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePodLogic {
	return &CreatePodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreatePodLogic) CreatePod(in *podengine.CreatePodReq) (*podengine.CreatePodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if len(in.Spec.Containers) == 0 {
		return nil, fmt.Errorf("pod spec must have at least one container")
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	containerSpec := in.Spec.Containers[0]

	config := &container.Config{
		Image:        containerSpec.Image,
		Env:          dockerx.BuildEnvList(containerSpec.Env),
		Cmd:          containerSpec.Args,
		Labels:       in.Spec.Labels,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Tty:          false,
		OpenStdin:    false,
		StdinOnce:    false,
	}

	networkMode := in.Spec.NetworkMode
	if networkMode == "" {
		networkMode = "bridge"
	}

	var portBindings nat.PortMap
	if networkMode != "host" && networkMode != "none" {
		portBindings = parsePorts(containerSpec.Ports)
	}

	// Parse resource limits
	resources := parseResources(containerSpec.Resources)

	// Parse volume mounts
	mounts := parseVolumeMounts(containerSpec.VolumeMounts)

	// Set termination grace period
	terminationGracePeriodSeconds := int(in.Spec.TerminationGracePeriodSeconds)
	if terminationGracePeriodSeconds <= 0 {
		terminationGracePeriodSeconds = 60 // Default 10 seconds
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		RestartPolicy: parseRestartPolicy(in.Spec.RestartPolicy),
		AutoRemove:    false,
		NetworkMode:   container.NetworkMode(networkMode),
		Privileged:    false,
		Resources:     resources,
		Mounts:        mounts,
	}

	// Set termination grace period in config
	config.StopTimeout = &terminationGracePeriodSeconds

	networkConfig := &network.NetworkingConfig{}

	if in.Spec.NetworkName != "" {
		hostConfig.NetworkMode = container.NetworkMode(in.Spec.NetworkName)
	}

	containerName := fmt.Sprintf("%s-%s", in.Name, strings.ToLower(containerSpec.Name))
	resp, err := dockerClient.ContainerCreate(l.ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		l.Errorf("Failed to create container: %v", err)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Inspect the container to get information
	containerInfo, err := dockerClient.ContainerInspect(l.ctx, resp.ID)
	if err != nil {
		l.Errorf("Failed to inspect container %s: %v", resp.ID, err)
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	pod := &podengine.Pod{
		Id:    containerInfo.ID,
		Name:  in.Name,
		Phase: podengine.PodPhase_POD_PHASE_PENDING,
		Containers: []*podengine.Container{
			{
				Name:  containerSpec.Name,
				Image: containerSpec.Image,
				State: &podengine.ContainerState{
					Running:      false,
					Terminated:   false,
					Waiting:      true,
					Reason:       containerInfo.State.Status,
					Message:      "Container created but not started",
					StartedTime:  "",
					FinishedTime: "",
					ExitCode:     "0",
				},
				Env:          containerSpec.Env,
				Args:         containerSpec.Args,
				Resources:    containerSpec.Resources,
				VolumeMounts: containerSpec.VolumeMounts,
			},
		},
		Labels:       in.Spec.Labels,
		Annotations:  in.Spec.Annotations,
		CreationTime: carbon.Parse(containerInfo.Created).ToDateTimeString(),
		StartTime:    "",
	}

	return &podengine.CreatePodRes{
		Pod: pod,
	}, nil
}

func parsePorts(ports []string) nat.PortMap {
	portMap := make(nat.PortMap)
	for _, port := range ports {
		parts := strings.Split(port, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPort := parts[1]
			portMap[nat.Port(containerPort+"/tcp")] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			}
		}
	}
	return portMap
}

func parseRestartPolicy(policy string) container.RestartPolicy {
	switch strings.ToLower(policy) {
	case "always":
		return container.RestartPolicy{
			Name: "always",
		}
	case "onfailure":
		return container.RestartPolicy{
			Name: "on-failure",
		}
	default:
		return container.RestartPolicy{
			Name: "no",
		}
	}
}

func parseResources(resourceMap map[string]string) container.Resources {
	resources := container.Resources{}

	if cpuLimit, ok := resourceMap["cpu"]; ok {
		cpuQuota := parseCpuLimit(cpuLimit)
		if cpuQuota > 0 {
			resources.CPUQuota = cpuQuota
			resources.CPUPeriod = 100000 // 100ms period
		}
	}

	if memoryLimit, ok := resourceMap["memory"]; ok {
		memoryBytes := parseMemoryLimit(memoryLimit)
		if memoryBytes > 0 {
			resources.Memory = memoryBytes
		}
	}

	if cpuRequest, ok := resourceMap["cpuRequest"]; ok {
		cpuShares := parseCpuShares(cpuRequest)
		if cpuShares > 0 {
			resources.CPUShares = cpuShares
		}
	}

	if memoryRequest, ok := resourceMap["memoryRequest"]; ok {
		memoryBytes := parseMemoryLimit(memoryRequest)
		if memoryBytes > 0 {
			resources.MemoryReservation = memoryBytes
		}
	}

	return resources
}

func parseCpuLimit(cpuLimit string) int64 {
	var cpu float64
	_, err := fmt.Sscanf(cpuLimit, "%f", &cpu)
	if err != nil {
		return 0
	}
	return int64(cpu * 100000)
}

func parseMemoryLimit(memoryLimit string) int64 {
	var value int64
	var unit string
	_, err := fmt.Sscanf(memoryLimit, "%d%s", &value, &unit)
	if err != nil {
		_, err = fmt.Sscanf(memoryLimit, "%d", &value)
		if err != nil {
			return 0
		}
		return value
	}
	switch strings.ToLower(unit) {
	case "k", "kb":
		return value * 1024
	case "m", "mb":
		return value * 1024 * 1024
	case "g", "gb":
		return value * 1024 * 1024 * 1024
	case "t", "tb":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		return value
	}
}

func parseCpuShares(cpuRequest string) int64 {
	var cpu float64
	_, err := fmt.Sscanf(cpuRequest, "%f", &cpu)
	if err != nil {
		return 0
	}
	return int64(cpu * 1024)
}

func parseVolumeMounts(volumeMounts []string) []mount.Mount {
	var mounts []mount.Mount
	for _, mountStr := range volumeMounts {
		parts := strings.Split(mountStr, ":")
		if len(parts) >= 2 {
			hostPath := parts[0]
			containerPath := parts[1]
			readOnly := false
			if len(parts) == 3 && parts[2] == "ro" {
				readOnly = true
			}
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   hostPath,
				Target:   containerPath,
				ReadOnly: readOnly,
			})
		}
	}
	return mounts
}
