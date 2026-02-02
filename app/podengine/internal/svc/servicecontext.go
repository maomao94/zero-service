package svc

import (
	"zero-service/app/podengine/internal/config"
	"zero-service/common/dockerx"

	"github.com/docker/docker/client"
)

type ServiceContext struct {
	Config       config.Config
	DockerClient *client.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	dockerClient := dockerx.MustNewClient(client.FromEnv, client.WithAPIVersionNegotiation())
	return &ServiceContext{
		Config:       c,
		DockerClient: dockerClient,
	}
}
