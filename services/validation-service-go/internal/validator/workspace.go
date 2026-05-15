package validator

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"regexp"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	dockerfileFromRegex   = regexp.MustCompile(`(?i)^\s*FROM\s+`)
	dockerfileExposeRegex = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(\d+)`)
	cmakeProjectRegex     = regexp.MustCompile(`(?i)^\s*project\s*\(`)
	cmakeStandardRegex    = regexp.MustCompile(`(?i)^\s*set\s*\(\s*CMAKE_CXX_STANDARD\s+(\d+)`)
	cmakeExecutableRegex  = regexp.MustCompile(`(?i)^\s*add_executable\s*\(`)
	mainFunctionRegex     = regexp.MustCompile(`(?m)^int\s+main\s*\(`)
)

type CMakeInfo struct {
	Exists          bool
	HasProject      bool
	HasStandard     bool
	HasExecutable   bool
	StandardVersion string
}

type DockerfileInfo struct {
	Exists    bool
	HasFROM   bool
	HasEXPOSE bool
	Port      int
	BaseImage string
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
	Docker DockerfileInfo
	Main   MainInfo

	MissingDirs       []string
	MissingFiles      []string
	ExtensionCounts   map[string]int
	ForbiddenErrors   []domain.ValidationError
	ForbiddenWarnings []domain.ValidationError
	PortFoundInSrc    bool
}

// AnalyzeWorkspace performs a single pass over the submission files.
func AnalyzeWorkspace(rootDir string, contract *domain.SubmissionContract) *WorkspaceContext {
	ctx := &WorkspaceContext{
		RootDir:         rootDir,
		Contract:        contract,
		ExtensionCounts: make(map[string]int),
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

	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)

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

		// Port search in src/
		if !ctx.PortFoundInSrc && portStr != "" && strings.HasPrefix(relPath, "src/") {
			// Read text content
			content, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(content), portStr) {
				ctx.PortFoundInSrc = true
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

		if dockerfileFromRegex.MatchString(line) {
			ctx.Docker.HasFROM = true
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ctx.Docker.BaseImage = parts[1]
			}
		}

		if matches := dockerfileExposeRegex.FindStringSubmatch(line); len(matches) >= 2 {
			ctx.Docker.HasEXPOSE = true
			if port, err := strconv.Atoi(matches[1]); err == nil {
				ctx.Docker.Port = port
			}
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

		if cmakeProjectRegex.MatchString(line) {
			ctx.CMake.HasProject = true
		}
		if matches := cmakeStandardRegex.FindStringSubmatch(line); len(matches) >= 2 {
			ctx.CMake.HasStandard = true
			ctx.CMake.StandardVersion = matches[1]
		}
		if cmakeExecutableRegex.MatchString(line) {
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

	contentStr := string(content)
	if mainFunctionRegex.MatchString(contentStr) || strings.Contains(contentStr, "int main(") {
		ctx.Main.HasMain = true
	}

	for _, pattern := range ctx.Contract.EndpointPatterns {
		if strings.Contains(contentStr, pattern) {
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
