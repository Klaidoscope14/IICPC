package validator

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	workspaceDockerfileFromRegex   = regexp.MustCompile(`(?i)^\s*FROM\s+`)
	workspaceDockerfileExposeRegex = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(.+)$`)
	workspaceDockerfileUserRegex   = regexp.MustCompile(`(?i)^\s*USER\s+`)
	workspaceDockerfileHealthRegex = regexp.MustCompile(`(?i)^\s*HEALTHCHECK\s+`)
	workspaceCMakeProjectRegex     = regexp.MustCompile(`(?i)^\s*project\s*\(`)
	workspaceCMakeStandardRegex    = regexp.MustCompile(`(?i)^\s*set\s*\(\s*CMAKE_CXX_STANDARD\s+(\d+)`)
	workspaceCMakeExecutableRegex  = regexp.MustCompile(`(?i)^\s*add_executable\s*\(`)
	workspaceMainFunctionRegex     = regexp.MustCompile(`(?m)^int\s+main\s*\(`)
)

type CMakeInfo struct {
	Exists          bool
	HasProject      bool
	HasStandard     bool
	HasExecutable   bool
	StandardVersion string
}

type WorkspaceDockerfileInfo struct {
	Exists         bool
	HasFROM        bool
	HasEXPOSE      bool
	HasUSER        bool
	HasHEALTHCHECK bool
	Port           int
	ExposedPorts   []int
	BaseImage      string
}

type MainInfo struct {
	Exists      bool
	HasMain     bool
	HasEndpoint bool
}

type WorkspaceContext struct {
	RootDir  string
	Contract *domain.SubmissionContract

	CMake  CMakeInfo
	Docker WorkspaceDockerfileInfo
	Main   MainInfo

	MissingDirs       []string
	MissingFiles      []string
	ExtensionCounts   map[string]int
	ForbiddenErrors   []domain.ValidationError
	ForbiddenWarnings []domain.ValidationError
	PortFoundInSrc    bool
	PortsFoundInSrc   map[int]bool
	EndpointSignals   EndpointSignals
}

type EndpointSignals struct {
	NetworkPattern       bool
	MethodHits           map[string]bool
	PathHits             map[string]bool
	SchemaFieldHits      map[string]bool
	WebSocketPathHits    map[string]bool
	WebSocketMessageHits map[string]bool
	WebSocketUpgrade     bool
	PingHandler          bool
}

// AnalyzeWorkspace performs a single pass over the submission files.
func AnalyzeWorkspace(rootDir string, contract *domain.SubmissionContract) *WorkspaceContext {
	ctx := &WorkspaceContext{
		RootDir:         rootDir,
		Contract:        contract,
		ExtensionCounts: make(map[string]int),
		PortsFoundInSrc: make(map[int]bool),
		EndpointSignals: EndpointSignals{
			MethodHits:           make(map[string]bool),
			PathHits:             make(map[string]bool),
			SchemaFieldHits:      make(map[string]bool),
			WebSocketPathHits:    make(map[string]bool),
			WebSocketMessageHits: make(map[string]bool),
		},
	}

	// 1. Direct reads (O(1) lookups instead of walking)
	ctx.analyzeDockerfile()
	ctx.analyzeCMake()
	ctx.analyzeMain()

	// 2. Check required directories and files
	ctx.checkExistence()

	// 3. Single WalkDir for extensions, patterns, and port searching
	portStr := ""
	if ctx.Docker.HasEXPOSE {
		portStr = strconv.Itoa(ctx.Docker.Port)
	}
	portStrings := make(map[int]string)
	for _, port := range append(ctx.Docker.ExposedPorts, contract.RuntimeAPI.RequiredPorts...) {
		if port > 0 {
			portStrings[port] = strconv.Itoa(port)
		}
	}

	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		relPath = filepath.ToSlash(relPath)

		// Check forbidden patterns
		for _, pattern := range contract.ForbiddenPatterns {
			if strings.Contains(relPath, pattern) {
				ctx.ForbiddenWarnings = append(ctx.ForbiddenWarnings, domain.ValidationError{
					Code:     "FORBIDDEN_PATTERN",
					Message:  "File matches forbidden pattern '" + pattern + "'",
					Severity: domain.SeverityWarning,
					FilePath: relPath,
				})
				return nil
			}
		}

		// Extensions check & count
		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			ctx.ExtensionCounts[ext]++
		}

		if contract.ForbiddenExtensions[ext] {
			ctx.ForbiddenErrors = append(ctx.ForbiddenErrors, domain.ValidationError{
				Code:     "FORBIDDEN_EXTENSION",
				Message:  "File has forbidden extension: " + ext,
				Severity: domain.SeverityError,
				FilePath: relPath,
			})
			return nil // Don't bother scanning forbidden files
		}
		if len(contract.AllowedExtensions) > 0 && !contract.AllowedExtensions[ext] {
			ctx.ForbiddenWarnings = append(ctx.ForbiddenWarnings, domain.ValidationError{
				Code:     "UNSUPPORTED_EXTENSION",
				Message:  "File extension is not part of the submission contract: " + ext,
				Severity: domain.SeverityWarning,
				FilePath: relPath,
			})
		}

		if isContractSourceFile(relPath) {
			info, err := d.Info()
			if err == nil && info.Size() <= 1024*1024 { // 1MB cap
				content, err := os.ReadFile(path)
				if err == nil {
					lowerContent := bytes.ToLower(content)
					if !ctx.PortFoundInSrc && portStr != "" && bytes.Contains(content, []byte(portStr)) {
						ctx.PortFoundInSrc = true
					}
					for port, value := range portStrings {
						if bytes.Contains(content, []byte(value)) {
							ctx.PortsFoundInSrc[port] = true
						}
					}
					ctx.collectEndpointSignals(lowerContent)
				}
			}
		}

		return nil
	})

	return ctx
}

func (ctx *WorkspaceContext) analyzeDockerfile() {
	path := filepath.Join(ctx.RootDir, "Dockerfile")
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	ctx.Docker.Exists = true

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if workspaceDockerfileFromRegex.MatchString(line) {
			ctx.Docker.HasFROM = true
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ctx.Docker.BaseImage = parts[1]
			}
		}

		if matches := workspaceDockerfileExposeRegex.FindStringSubmatch(line); len(matches) >= 2 {
			ctx.Docker.HasEXPOSE = true
			for _, token := range strings.Fields(matches[1]) {
				if port, ok := parseExposePort(token); ok {
					ctx.Docker.ExposedPorts = append(ctx.Docker.ExposedPorts, port)
					if ctx.Docker.Port == 0 {
						ctx.Docker.Port = port
					}
				}
			}
		}
		if workspaceDockerfileUserRegex.MatchString(line) {
			ctx.Docker.HasUSER = true
		}
		if workspaceDockerfileHealthRegex.MatchString(line) {
			ctx.Docker.HasHEALTHCHECK = true
		}
	}
}

func (ctx *WorkspaceContext) analyzeCMake() {
	path := filepath.Join(ctx.RootDir, "CMakeLists.txt")
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	ctx.CMake.Exists = true

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if workspaceCMakeProjectRegex.MatchString(line) {
			ctx.CMake.HasProject = true
		}
		if matches := workspaceCMakeStandardRegex.FindStringSubmatch(line); len(matches) >= 2 {
			ctx.CMake.HasStandard = true
			ctx.CMake.StandardVersion = matches[1]
		}
		if workspaceCMakeExecutableRegex.MatchString(line) {
			ctx.CMake.HasExecutable = true
		}
	}
}

func (ctx *WorkspaceContext) analyzeMain() {
	path := filepath.Join(ctx.RootDir, "src", "main.cpp")
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	ctx.Main.Exists = true

	if workspaceMainFunctionRegex.Match(content) || bytes.Contains(content, []byte("int main(")) {
		ctx.Main.HasMain = true
	}

	for _, pattern := range ctx.Contract.RuntimeAPI.EndpointPatterns {
		if bytes.Contains(content, []byte(pattern)) {
			ctx.Main.HasEndpoint = true
			break
		}
	}
}

func (ctx *WorkspaceContext) checkExistence() {
	for _, dir := range ctx.Contract.RequiredDirs {
		info, err := os.Stat(filepath.Join(ctx.RootDir, dir))
		if err != nil || !info.IsDir() {
			ctx.MissingDirs = append(ctx.MissingDirs, dir)
		}
	}

	for _, file := range ctx.Contract.RequiredFiles {
		info, err := os.Stat(filepath.Join(ctx.RootDir, file))
		if err != nil || info.IsDir() {
			ctx.MissingFiles = append(ctx.MissingFiles, file)
		}
	}
}

func (ctx *WorkspaceContext) collectEndpointSignals(lowerContent []byte) {
	for _, pattern := range ctx.Contract.RuntimeAPI.EndpointPatterns {
		if pattern != "" && bytes.Contains(lowerContent, []byte(strings.ToLower(pattern))) {
			ctx.EndpointSignals.NetworkPattern = true
			ctx.Main.HasEndpoint = true
			break
		}
	}

	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		lowerMethod := []byte(strings.ToLower(method))
		if bytes.Contains(lowerContent, lowerMethod) {
			ctx.EndpointSignals.MethodHits[method] = true
		}
	}

	endpoints := []domain.EndpointSpec{
		ctx.Contract.RuntimeAPI.HealthEndpoint,
		ctx.Contract.RuntimeAPI.OrderEndpoint,
		ctx.Contract.RuntimeAPI.CancelEndpoint,
	}
	for _, endpoint := range endpoints {
		if endpoint.Path == "" {
			continue
		}
		if endpointPathPresent(lowerContent, endpoint.Path) {
			ctx.EndpointSignals.PathHits[endpoint.Path] = true
		}
		for field := range endpoint.Schema {
			if bytes.Contains(lowerContent, []byte(strings.ToLower(field))) {
				ctx.EndpointSignals.SchemaFieldHits[field] = true
			}
		}
	}
	if bytes.Contains(lowerContent, []byte("content-type")) || bytes.Contains(lowerContent, []byte("application/json")) {
		ctx.EndpointSignals.SchemaFieldHits["content-type"] = true
	}

	ws := ctx.Contract.RuntimeAPI.MarketDataStream
	if ws.Path != "" && endpointPathPresent(lowerContent, ws.Path) {
		ctx.EndpointSignals.WebSocketPathHits[ws.Path] = true
	}
	for _, msgType := range ws.MessageTypes {
		if bytes.Contains(lowerContent, []byte(strings.ToLower(msgType))) {
			ctx.EndpointSignals.WebSocketMessageHits[msgType] = true
		}
	}
	if bytes.Contains(lowerContent, []byte("websocket")) || bytes.Contains(lowerContent, []byte("upgrade")) {
		ctx.EndpointSignals.WebSocketUpgrade = true
	}
	if bytes.Contains(lowerContent, []byte("ping")) || bytes.Contains(lowerContent, []byte("pong")) || bytes.Contains(lowerContent, []byte("heartbeat")) {
		ctx.EndpointSignals.PingHandler = true
	}
}

func isContractSourceFile(relPath string) bool {
	if strings.HasPrefix(relPath, "src/") || strings.HasPrefix(relPath, "include/") {
		return true
	}
	switch filepath.Base(relPath) {
	case "CMakeLists.txt", "Dockerfile":
		return true
	}
	return false
}

func parseExposePort(token string) (int, bool) {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, `"`)
	if token == "" {
		return 0, false
	}
	if idx := strings.Index(token, "/"); idx >= 0 {
		token = token[:idx]
	}
	port, err := strconv.Atoi(token)
	if err != nil {
		return 0, false
	}
	return port, true
}

func endpointPathPresent(lowerContent []byte, path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	for _, alias := range endpointPathAliases(path) {
		if bytes.Contains(lowerContent, []byte(alias)) {
			return true
		}
	}
	return false
}

func endpointPathAliases(path string) []string {
	aliases := []string{path}
	if strings.Contains(path, "{") {
		base := path
		if idx := strings.Index(base, "{"); idx >= 0 {
			base = strings.TrimRight(base[:idx], "/")
		}
		if base != "" {
			aliases = append(aliases, base)
		}
	}
	aliases = append(aliases, strings.ReplaceAll(path, "/", "\\/"))
	return aliases
}
