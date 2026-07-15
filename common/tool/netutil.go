package tool

import (
	"os"
	"strings"
)

// MayReplaceLocalhost maps localhost addresses to the Docker host when running in Docker.
func MayReplaceLocalhost(host string) string {
	if os.Getenv("IS_DOCKER") != "" {
		return strings.Replace(strings.Replace(host,
			"localhost", "host.docker.internal", 1),
			"127.0.0.1", "host.docker.internal", 1)
	}
	return host
}
