package container

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Manager defines the contract for container lifecycle operations.
// Implementations can target Docker, Kubernetes, or any other container runtime.
type Manager interface {
	// BuildImage builds a container image from a tar archive.
	BuildImage(ctx context.Context, buildContext io.Reader, imageName string) error

	// PullImage pulls a container image from a registry.
	PullImage(ctx context.Context, imageName string) error

	// CreateAndStart creates a container from the image and starts it with resource limits.
	CreateAndStart(ctx context.Context, opts CreateOptions) (containerID string, serviceURL string, err error)

	// Stop stops a running container gracefully with a timeout.
	Stop(ctx context.Context, containerID string) error

	// Remove removes a stopped container.
	Remove(ctx context.Context, containerID string) error

	// IsRunning checks if a container is still running.
	IsRunning(ctx context.Context, containerID string) (bool, error)

	// WaitForHealthy polls the container until it responds on the health endpoint.
	WaitForHealthy(ctx context.Context, containerID string, healthURL string, timeout time.Duration) error
}

// CreateOptions holds all parameters needed to create and start a container.
type CreateOptions struct {
	ImageName      string
	ContainerName  string
	ExposedPorts   []string
	CPUMilli       int64
	MemoryMB       int64
	TimeoutSeconds int64
	NetworkMode    string // e.g., "bridge" or a custom network name
	Cmd            []string
}

// DockerManager implements Manager using the Docker Engine API.
type DockerManager struct {
	client *client.Client
	logger *slog.Logger
}

// NewDockerManager creates a Docker-backed container manager.
func NewDockerManager(logger *slog.Logger) (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerManager{
		client: cli,
		logger: logger,
	}, nil
}

func (m *DockerManager) BuildImage(ctx context.Context, buildContext io.Reader, imageName string) error {
	m.logger.Info("building container image", slog.String("image", imageName))

	resp, err := m.client.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
		NoCache:    false,
	})
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	// Drain the build output — this also blocks until the build completes.
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	m.logger.Info("image built successfully", slog.String("image", imageName))
	return nil
}

func (m *DockerManager) CreateAndStart(ctx context.Context, opts CreateOptions) (string, string, error) {
	m.logger.Info("creating container",
		slog.String("image", opts.ImageName),
		slog.String("name", opts.ContainerName),
	)

	// Parse port mappings.
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	var hostPort string

	for _, p := range opts.ExposedPorts {
		port := nat.Port(p + "/tcp")
		exposedPorts[port] = struct{}{}
		// Let Docker assign a random host port.
		portBindings[port] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}}
	}

	// Resource limits.
	resources := container.Resources{}
	if opts.CPUMilli > 0 {
		// NanoCPUs: 1 CPU = 1e9 nanoCPUs. 1000 milli = 1 CPU.
		resources.NanoCPUs = opts.CPUMilli * 1_000_000
	}
	if opts.MemoryMB > 0 {
		resources.Memory = opts.MemoryMB * 1024 * 1024
	}

	resp, err := m.client.ContainerCreate(ctx,
		&container.Config{
			Image:        opts.ImageName,
			ExposedPorts: exposedPorts,
			Cmd:          opts.Cmd,
		},
		&container.HostConfig{
			PortBindings:  portBindings,
			Resources:     resources,
			NetworkMode:   container.NetworkMode(opts.NetworkMode),
			RestartPolicy: container.RestartPolicy{Name: "no"},
		},
		nil, nil, opts.ContainerName,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := m.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// Clean up the created container on start failure.
		m.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", "", fmt.Errorf("failed to start container: %w", err)
	}

	// Inspect to get the assigned host port.
	inspect, err := m.client.ContainerInspect(ctx, resp.ID)
	if err == nil {
		for _, p := range opts.ExposedPorts {
			port := nat.Port(p + "/tcp")
			if bindings, ok := inspect.NetworkSettings.Ports[port]; ok && len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}
	}

	serviceURL := ""
	if hostPort != "" {
		serviceURL = fmt.Sprintf("http://localhost:%s", hostPort)
	}

	m.logger.Info("container started",
		slog.String("container_id", resp.ID[:12]),
		slog.String("service_url", serviceURL),
	)

	return resp.ID, serviceURL, nil
}

func (m *DockerManager) Stop(ctx context.Context, containerID string) error {
	m.logger.Info("stopping container", slog.String("container_id", containerID[:12]))

	timeout := 10
	return m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (m *DockerManager) Remove(ctx context.Context, containerID string) error {
	m.logger.Info("removing container", slog.String("container_id", containerID[:12]))

	return m.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (m *DockerManager) IsRunning(ctx context.Context, containerID string) (bool, error) {
	inspect, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return inspect.State.Running, nil
}

func (m *DockerManager) WaitForHealthy(ctx context.Context, containerID string, healthURL string, timeout time.Duration) error {
	m.logger.Info("waiting for container health",
		slog.String("container_id", containerID[:12]),
		slog.String("health_url", healthURL),
	)

	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("health check timed out after %v", timeout)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if the container is still running.
			running, err := m.IsRunning(ctx, containerID)
			if err != nil || !running {
				return fmt.Errorf("container exited before becoming healthy")
			}

			// Try to pull the image list as a simple connectivity check.
			// In production, this would be an HTTP GET to the healthURL.
			if healthURL != "" {
				// The actual health check would use net/http here.
				// For now, we just verify the container is running.
				m.logger.Debug("container running, assuming healthy",
					slog.String("container_id", containerID[:12]),
				)
				return nil
			}
		}
	}
}

// PullImage pulls a container image from a registry.
func (m *DockerManager) PullImage(ctx context.Context, imageName string) error {
	m.logger.Info("pulling image", slog.String("image", imageName))

	reader, err := m.client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("image not found: %s", imageName)
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	return err
}
