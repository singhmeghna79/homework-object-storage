package dockerClient

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const s3MinioApiPort = "9000"

// MinioNode represents a discovered Minio node
type MinioNode struct {
	ID        string
	Name      string
	IPAddress string
	Port      string
	AccessKey string
	SecretKey string
}

// DockerClient wraps the Docker API client
type DockerClient struct {
	client *client.Client
}

// NewClient creates a new Docker client
func NewClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &DockerClient{client: cli}, nil
}

// DiscoverMinioNodes finds all running Minio nodes
func (d *DockerClient) DiscoverMinioNodes(ctx context.Context) ([]MinioNode, error) {
	// Create filter to find Minio containers
	filter := filters.NewArgs()
	filter.Add("name", "^/amazin-object-storage-node$")

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		// Filters: filter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var nodes []MinioNode
	for _, container := range containers {
		if !strings.Contains(container.Names[0], "amazin-object-storage-node") {
			continue
		}

		// Get container details to extract environment variables
		inspect, err := d.client.ContainerInspect(ctx, container.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container %s: %w", container.ID, err)
		}
		var accessKey, secretKey string
		for _, env := range inspect.Config.Env {
			if strings.HasPrefix(env, "MINIO_ACCESS_KEY=") {
				accessKey = strings.TrimPrefix(env, "MINIO_ACCESS_KEY=")
			} else if strings.HasPrefix(env, "MINIO_SECRET_KEY=") {
				secretKey = strings.TrimPrefix(env, "MINIO_SECRET_KEY=")
			}
		}

		// Get IP address from the object storage network
		var ipAddress string
		if networks := inspect.NetworkSettings.Networks; networks != nil {
			if network, ok := networks[container.HostConfig.NetworkMode]; ok {
				ipAddress = network.IPAddress
			}
		}
		if ipAddress == "" {
			continue // Skip containers without a valid IP address
		}

		// var mappedPort string
		// if bindings, ok := inspect.NetworkSettings.Ports["9001/tcp"]; ok && len(bindings) > 0 {
		// 	mappedPort = bindings[0].HostPort
		// } else if bindings, ok := inspect.NetworkSettings.Ports["9000/tcp"]; ok && len(bindings) > 0 {
		// 	mappedPort = bindings[0].HostPort
		// }

		// if mappedPort == "" {
		// 	log.Printf("No mapped MinIO port found for container: %s\n", container.Names[0])
		// 	continue
		// }

		// Default Minio API port is 9000
		node := MinioNode{
			ID:        container.ID,
			Name:      strings.TrimPrefix(container.Names[0], "/"),
			IPAddress: ipAddress,
			Port:      s3MinioApiPort, // Minio s3 API port
			AccessKey: accessKey,
			SecretKey: secretKey,
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// ValidateNodeConnection checks if a node is accessible
func (d *DockerClient) ValidateNodeConnection(node MinioNode) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", node.IPAddress, node.Port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
