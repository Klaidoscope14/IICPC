package domain

// SubmissionContract defines the structural and content requirements
// for a valid trade-engine submission. All validators check against this contract.
//
// The contract is derived from the reference trade-engine/ directory.
var DefaultContract = SubmissionContract{
	RequiredDirs: []string{
		"src",
		"include",
	},
	RequiredFiles: []string{
		"CMakeLists.txt",
		"Dockerfile",
		"src/main.cpp",
	},
	AllowedExtensions: map[string]bool{
		".cpp":   true,
		".c":     true,
		".h":     true,
		".hpp":   true,
		".txt":   true, // CMakeLists.txt
		".cmake": true,
		".md":    true,
		".json":  true,
		".yaml":  true,
		".yml":   true,
		".toml":  true,
		".cfg":   true,
		".conf":  true,
		".proto": true,
		".py":    true, // Build scripts
		".sh":    true, // Build scripts (allowed but warned)
		"":       true, // Files without extensions (Makefile, Dockerfile, etc.)
	},
	ForbiddenExtensions: map[string]bool{
		".exe":   true,
		".dll":   true,
		".so":    true,
		".dylib": true,
		".o":     true,
		".obj":   true,
		".a":     true,
		".lib":   true,
		".class": true,
		".jar":   true,
		".war":   true,
		".pyc":   true,
		".pyo":   true,
		".bat":   true,
		".com":   true,
		".bin":   true,
	},
	ForbiddenPatterns: []string{
		".git/",
		".svn/",
		"__pycache__/",
		"node_modules/",
		".DS_Store",
	},
	DockerfileRequirements: DockerfileRequirements{
		RequireFROM:   true,
		RequireEXPOSE: true,
	},
	CMakeRequirements: CMakeRequirements{
		RequireProject:       true,
		RequireCXXStandard:   true,
		RequireAddExecutable: true,
	},
	MaxExtractedBytes: 500 * 1024 * 1024, // 500 MB
	MaxFileCount:      1000,
	// Common server-socket patterns in C++ that indicate the submission
	// can accept network connections (required for bot fleet benchmarking).
	EndpointPatterns: []string{
		"bind",
		"listen",
		"accept",
		"httplib",
		"boost::asio",
		"crow::",
		"pistache",
		"restbed",
		"cpprestsdk",
		"grpc::",
		"uWebSockets",
	},
}

// SubmissionContract holds all validation rules.
type SubmissionContract struct {
	RequiredDirs          []string
	RequiredFiles         []string
	AllowedExtensions     map[string]bool
	ForbiddenExtensions   map[string]bool
	ForbiddenPatterns     []string
	DockerfileRequirements DockerfileRequirements
	CMakeRequirements     CMakeRequirements
	MaxExtractedBytes     int64
	MaxFileCount          int
	EndpointPatterns      []string
}

// DockerfileRequirements defines what must be present in the Dockerfile.
type DockerfileRequirements struct {
	RequireFROM   bool
	RequireEXPOSE bool
}

// CMakeRequirements defines what must be present in CMakeLists.txt.
type CMakeRequirements struct {
	RequireProject       bool
	RequireCXXStandard   bool
	RequireAddExecutable bool
}
