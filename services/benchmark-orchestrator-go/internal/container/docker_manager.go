package container

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	units "github.com/docker/go-units"
)

// Manager defines the contract for container lifecycle operations.
// Implementations can target Docker, Kubernetes, or any other container runtime.
type Manager interface {
	// BuildImage builds a container image from a tar archive.
	BuildImage(ctx context.Context, buildContext io.Reader, imageName string) (BuildResult, error)

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

	// CaptureLogs streams stdout and stderr from the container into the provided logger.
	CaptureLogs(ctx context.Context, containerID string, logger *slog.Logger) error
}

// BuildResult contains bounded output collected from the Docker build stream.
type BuildResult struct {
	Logs      string
	Truncated bool
	Error     string
}

// CreateOptions holds all parameters needed to create and start a container.
type CreateOptions struct {
	ImageName      string
	ContainerName  string
	ExposedPorts   []string
	CPUMilli       int64
	MemoryMB       int64
	PidsLimit      int64
	TimeoutSeconds int64
	NetworkMode    string // e.g., "bridge" or a custom network name
	HostBindIP     string // Host interface for published ports.
	ServiceHost    string // Hostname clients should use to reach published ports.
	Cmd            []string
	RunAsUser      string
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

func (m *DockerManager) BuildImage(ctx context.Context, buildContext io.Reader, imageName string) (BuildResult, error) {
	m.logger.Info("building container image", slog.String("image", imageName))

	resp, err := m.client.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
		NoCache:    false,
	})
	if err != nil {
		return BuildResult{}, fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	result, err := readBuildStream(resp.Body)
	if err != nil {
		return result, fmt.Errorf("error reading build output: %w", err)
	}
	if result.Error != "" {
		return result, fmt.Errorf("docker build failed: %s", result.Error)
	}

	m.logger.Info("image built successfully", slog.String("image", imageName))
	return result, nil
}

func (m *DockerManager) CreateAndStart(ctx context.Context, opts CreateOptions) (string, string, error) {
	m.logger.Info("creating container",
		slog.String("image", opts.ImageName),
		slog.String("name", opts.ContainerName),
	)

	hostBindIP := strings.TrimSpace(opts.HostBindIP)
	if hostBindIP == "" {
		hostBindIP = "127.0.0.1"
	}
	serviceHost := strings.TrimSpace(opts.ServiceHost)
	if serviceHost == "" {
		serviceHost = "localhost"
	}
	networkMode := strings.TrimSpace(opts.NetworkMode)
	if networkMode == "" {
		networkMode = "bridge"
	}

	// Parse port mappings.
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	var hostPort string

	for _, p := range opts.ExposedPorts {
		port := nat.Port(p + "/tcp")
		exposedPorts[port] = struct{}{}
		// Let Docker assign a random host port.
		portBindings[port] = []nat.PortBinding{{HostIP: hostBindIP, HostPort: "0"}}
	}

	// Resource limits and Security Options for Sandboxing.
	resources := container.Resources{}
	if opts.CPUMilli > 0 {
		// NanoCPUs: 1 CPU = 1e9 nanoCPUs. 1000 milli = 1 CPU.
		resources.NanoCPUs = opts.CPUMilli * 1_000_000

		// Map 1000m CPUs to 1 physical core for pinning (simple heuristic for sandbox)
		cores := (opts.CPUMilli + 999) / 1000
		resources.CpusetCpus = fmt.Sprintf("0-%d", cores-1)
	}
	if opts.MemoryMB > 0 {
		resources.Memory = opts.MemoryMB * 1024 * 1024
		resources.MemorySwap = resources.Memory
	}
	pidsLimit := opts.PidsLimit
	if pidsLimit <= 0 {
		pidsLimit = 100
	}
	resources.PidsLimit = &pidsLimit
	resources.Ulimits = []*units.Ulimit{
		{Name: "nofile", Soft: 1024, Hard: 1024},
		{Name: "nproc", Soft: pidsLimit, Hard: pidsLimit},
	}

	runAsUser := opts.RunAsUser
	if runAsUser == "" {
		runAsUser = "65532:65532"
	}

	hostConfig := &container.HostConfig{
		PortBindings:   portBindings,
		PublishAllPorts: true,
		Resources:      resources,
		NetworkMode:    container.NetworkMode(networkMode),
		RestartPolicy:  container.RestartPolicy{Name: "no"},
		ReadonlyRootfs: true, // Read-only filesystem
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		Init:           boolPtr(true),
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=64m",
		},
	}

	containerConfig := &container.Config{
		Image: opts.ImageName,
		User:  runAsUser,
	}
	if len(exposedPorts) > 0 {
		containerConfig.ExposedPorts = exposedPorts
	}
	if len(opts.Cmd) > 0 {
		containerConfig.Cmd = opts.Cmd
	}

	resp, err := m.client.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
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
		if hostPort == "" {
			for _, bindings := range inspect.NetworkSettings.Ports {
				if len(bindings) > 0 && bindings[0].HostPort != "" {
					hostPort = bindings[0].HostPort
					break
				}
			}
		}
	}

	serviceURL := ""
	if hostPort != "" {
		serviceURL = fmt.Sprintf("http://%s:%s", serviceHost, hostPort)
	}

	m.logger.Info("container started",
		slog.String("container_id", shortID(resp.ID, 12)),
		slog.String("service_url", serviceURL),
	)

	return resp.ID, serviceURL, nil
}

func boolPtr(value bool) *bool {
	return &value
}

func (m *DockerManager) Stop(ctx context.Context, containerID string) error {
	m.logger.Info("stopping container", slog.String("container_id", shortID(containerID, 12)))

	timeout := 10
	return m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (m *DockerManager) Remove(ctx context.Context, containerID string) error {
	m.logger.Info("removing container", slog.String("container_id", shortID(containerID, 12)))

	return m.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

// CaptureLogs tails the container logs and writes them to the provided logger.
func (m *DockerManager) CaptureLogs(ctx context.Context, containerID string, logger *slog.Logger) error {
	logs, err := m.client.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}

	go func() {
		defer logs.Close()
		buf := make([]byte, 8)
		for {
			_, err := io.ReadFull(logs, buf)
			if err != nil {
				if err != io.EOF {
					logger.Debug("container log stream ended with error", slog.String("error", err.Error()))
				}
				return
			}
			var size uint32
			size = uint32(buf[4])<<24 | uint32(buf[5])<<16 | uint32(buf[6])<<8 | uint32(buf[7])
			
			payload := make([]byte, size)
			if _, err := io.ReadFull(logs, payload); err != nil {
				return
			}
			
			// Docker multiplexes stdout and stderr using an 8-byte header.
			// buf[0] is 1 for stdout, 2 for stderr.
			logType := "stdout"
			if buf[0] == 2 {
				logType = "stderr"
			}
			
			msg := strings.TrimSpace(string(payload))
			if msg != "" {
				logger.Info(msg, slog.String("type", "container_runtime"), slog.String("stream", logType))
			}
		}
	}()

	return nil
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
		slog.String("container_id", shortID(containerID, 12)),
		slog.String("health_url", healthURL),
	)

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.C:
			return fmt.Errorf("health check timed out after %v", timeout)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if the container is still running.
			running, err := m.IsRunning(ctx, containerID)
			if err != nil || !running {
				return fmt.Errorf("container exited before becoming healthy")
			}

			if healthURL == "" {
				return fmt.Errorf("health URL is empty")
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
			if err != nil {
				return fmt.Errorf("invalid health URL %q: %w", healthURL, err)
			}

			resp, err := healthHTTPClient.Do(req)
			if err != nil {
				m.logger.Debug("container health probe failed",
					slog.String("container_id", shortID(containerID, 12)),
					slog.String("error", err.Error()),
				)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
				m.logger.Info("container health probe passed",
					slog.String("container_id", shortID(containerID, 12)),
					slog.Int("status", resp.StatusCode),
				)
				return nil
			}
			m.logger.Debug("container health probe returned non-success",
				slog.String("container_id", shortID(containerID, 12)),
				slog.Int("status", resp.StatusCode),
			)
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

const maxBuildLogBytes = 512 * 1024

var healthHTTPClient = &http.Client{Timeout: 1 * time.Second}

type buildStreamMessage struct {
	Stream      string `json:"stream"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	ErrorDetail *struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}

func readBuildStream(r io.Reader) (BuildResult, error) {
	var out limitedBuffer
	out.limit = maxBuildLogBytes
	var buildErr string

	decoder := json.NewDecoder(r)
	for {
		var msg buildStreamMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return BuildResult{Logs: out.String(), Truncated: out.truncated}, err
		}

		switch {
		case msg.Stream != "":
			out.WriteString(msg.Stream)
		case msg.Error != "":
			buildErr = msg.Error
			out.WriteString(msg.Error)
			out.WriteString("\n")
		case msg.Status != "":
			out.WriteString(msg.Status)
			out.WriteString("\n")
		}
		if msg.ErrorDetail != nil && msg.ErrorDetail.Message != "" {
			buildErr = msg.ErrorDetail.Message
		}
	}

	return BuildResult{Logs: out.String(), Truncated: out.truncated, Error: buildErr}, nil
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) WriteString(value string) {
	_, _ = b.Write([]byte(value))
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func shortID(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
