# Single-Pass Shared Context Refactoring

This plan outlines the architectural refactoring of the `internal/validator` package to eliminate I/O and processing redundancy. By introducing a shared `WorkspaceContext`, we consolidate file system traversals and file parsing into a single initialization step.

## Proposed Changes

### 1. Introduce `WorkspaceContext`

#### [NEW] [workspace.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/workspace.go)
Create a `WorkspaceContext` that acts as the single source of truth for the validation pipeline.

```go
type CMakeInfo struct {
    Exists bool
    HasProject bool
    HasStandard bool
    HasExecutable bool
    StandardVersion string
}

type DockerfileInfo struct {
    Exists bool
    HasFROM bool
    HasEXPOSE bool
    Port int
}

type MainInfo struct {
    Exists bool
    HasMain bool
    HasEndpoint bool
}

type WorkspaceContext struct {
    RootDir  string
    Contract *domain.SubmissionContract
    
    // Parsed File Info
    CMake  CMakeInfo
    Docker DockerfileInfo
    Main   MainInfo
    
    // Aggregated Walk Data
    MissingDirs       []string
    MissingFiles      []string
    ExtensionCounts   map[string]int
    ForbiddenErrors   []domain.ValidationError
    ForbiddenWarnings []domain.ValidationError
    PortFoundInSrc    bool
}
```
Add an `AnalyzeWorkspace(rootDir string, contract *domain.SubmissionContract) *WorkspaceContext` function that:
1. Directly reads and parses `Dockerfile` (to get the port early).
2. Directly reads and parses `CMakeLists.txt`.
3. Directly reads and parses `src/main.cpp`.
4. Performs a **single `filepath.WalkDir`** over `rootDir` to:
   - Check for forbidden patterns/extensions.
   - Aggregate file extension counts.
   - Check if the exposed port (from step 1) exists in any file under `src/`.
5. Checks existence of required files/directories without redundant `os.Stat` calls (using the walk data).

### 2. Update Validator Interface

#### [MODIFY] [pipeline.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/pipeline.go)
Update the `Pipeline` to initialize the `WorkspaceContext` and pass it to validators.
Modify the implicit Validator interface (or define an explicit one):
```go
type Validator interface {
    Name() string
    Validate(ctx *WorkspaceContext) domain.CheckResult
}
```

### 3. Simplify Individual Validators

Refactor all validators to consume `WorkspaceContext` instead of performing I/O:

#### [MODIFY] [folder_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/folder_validator.go)
Check `ctx.MissingDirs` and `ctx.MissingFiles`.

#### [MODIFY] [extension_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/extension_validator.go)
Return `ctx.ForbiddenErrors` and `ctx.ForbiddenWarnings`.

#### [MODIFY] [dockerfile_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/dockerfile_validator.go)
Check `ctx.Docker.Exists`, `ctx.Docker.HasFROM`, and `ctx.Docker.HasEXPOSE`.

#### [MODIFY] [config_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/config_validator.go)
Check `ctx.CMake.Exists`, `ctx.CMake.HasProject`, etc.

#### [MODIFY] [schema_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/schema_validator.go)
Check `ctx.Main.Exists`, `ctx.Main.HasMain`, and `ctx.Main.HasEndpoint`.

#### [MODIFY] [port_validator.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/port_validator.go)
Check `ctx.PortFoundInSrc` and validate `ctx.Docker.Port`.

#### [MODIFY] [language_detector.go](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/validation-service-go/internal/validator/language_detector.go)
Use `ctx.CMake.StandardVersion` or fallback to checking `ctx.ExtensionCounts`.

## Verification Plan
1. Ensure `go build ./...` passes in the workspace.
2. Ensure `go vet ./...` passes.
3. Verify that the logic remains exactly the same, but file I/O operations are drastically reduced.
