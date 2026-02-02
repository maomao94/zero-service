package dockerx

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.opentelemetry.io/otel"
)

func MustNewClient(ops ...client.Opt) *client.Client {
	ops = append(ops, client.WithTraceProvider(otel.GetTracerProvider()))
	cli, err := client.NewClientWithOpts(ops...)
	if err != nil {
		panic(err)
	}
	return cli
}

func ParseContainerEnv(env []string) map[string]string {
	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				key := e[:i]
				value := e[i+1:]
				envMap[key] = value
				break
			}
		}
	}
	return envMap
}

func ExtractContainerPorts(networkSettings *container.NetworkSettings) []string {
	var ports []string
	for port, bindings := range networkSettings.Ports {
		for _, binding := range bindings {
			ports = append(ports, fmt.Sprintf("%s->%s:%s", port.Port(), binding.HostIP, binding.HostPort))
		}
	}
	return ports
}

func ExtractContainerVolumeMounts(mounts []container.MountPoint) []string {
	var volumeMounts []string
	for _, mount := range mounts {
		mode := "rw"
		if !mount.RW {
			mode = "ro"
		}
		volumeMount := fmt.Sprintf("%s:%s:%s", mount.Source, mount.Destination, mode)
		volumeMounts = append(volumeMounts, volumeMount)
	}
	return volumeMounts
}

func ParseContainerResources(resources container.Resources) map[string]string {
	resourceMap := make(map[string]string)

	// Parse CPU limit
	if resources.CPUQuota > 0 {
		cpuLimit := float64(resources.CPUQuota) / 100000
		resourceMap["cpu"] = fmt.Sprintf("%f", cpuLimit)
	}

	// Parse memory limit
	if resources.Memory > 0 {
		memoryLimit := resources.Memory
		resourceMap["memory"] = fmt.Sprintf("%d", memoryLimit)
	}

	// Parse CPU shares (request)
	if resources.CPUShares > 0 {
		cpuRequest := float64(resources.CPUShares) / 1024
		resourceMap["cpuRequest"] = fmt.Sprintf("%f", cpuRequest)
	}

	// Parse memory reservation (request)
	if resources.MemoryReservation > 0 {
		memoryRequest := resources.MemoryReservation
		resourceMap["memoryRequest"] = fmt.Sprintf("%d", memoryRequest)
	}

	return resourceMap
}

func BuildEnvList(envMap map[string]string) []string {
	var envList []string
	for key, value := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
	}
	return envList
}
