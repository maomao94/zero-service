package svc

import (
	"sync"
	"zero-service/app/podengine/internal/config"
	"zero-service/common/dockerx"

	"github.com/docker/docker/client"
)

type ServiceContext struct {
	Config config.Config
	//DockerClient  *client.Client
	DockerClients map[string]*client.Client
	mu            sync.RWMutex
}

func NewServiceContext(c config.Config) *ServiceContext {
	dockerClients := make(map[string]*client.Client)

	dockerClient := dockerx.MustNewClient(
		client.FromEnv,
		client.WithAPIVersionNegotiation())
	dockerClients["local"] = dockerClient
	if c.DockerConfig != nil {
		for name, host := range c.DockerConfig {
			if name != "local" {
				hostClient := dockerx.MustNewClient(
					client.WithHost(host),
					client.WithAPIVersionNegotiation())
				dockerClients[name] = hostClient
			}
		}
	}
	return &ServiceContext{
		Config: c,
		//DockerClient:  dockerClient,
		DockerClients: dockerClients,
	}
}

func (svc *ServiceContext) GetDockerClient(name string) (*client.Client, bool) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	if len(name) == 0 || name == "local" {
		name = "local"
	}
	cli, ok := svc.DockerClients[name]
	return cli, ok
}
