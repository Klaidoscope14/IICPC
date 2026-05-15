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
			RequireFROM:        true,
			RequireEXPOSE:      true,
			WarnIfNoUSER:       true,
			WarnIfNoHEALTHCHECK: true,
		},
		CMakeRequirements: CMakeRequirements{
			RequireProject:       true,
			RequireCXXStandard:   true,
			RequireAddExecutable: true,
		},
		RuntimeAPI: RuntimeAPIContract{
			HealthEndpoint: EndpointSpec{
				Required: true,
				Method:   "GET",
				Path:     "/health",
			},
			OrderEndpoint: EndpointSpec{
				Required:    true,
				Method:      "POST",
				Path:        "/api/v1/orders",
				ContentType: "application/json",
				Schema: map[string]string{
					"id":       "string|integer",
					"symbol":   "string",
					"side":     "buy|sell",
					"price":    "number",
					"quantity": "integer",
					"type":     "limit|market",
				},
			},
			CancelEndpoint: EndpointSpec{
				Required:    true,
				Method:      "DELETE",
				Path:        "/api/v1/orders/{id}",
				ContentType: "application/json",
			},
			MarketDataStream: WebSocketSpec{
				Required:     true,
				Path:         "/ws/market-data",
				MessageTypes: []string{"book_snapshot", "trade", "heartbeat"},
				RequiresPing: true,
			},
			RequiredPorts: []int{8080},
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
				"websocket",
				"upgrade",
			},
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
	RuntimeAPI            RuntimeAPIContract
	MaxExtractedBytes     int64
	MaxFileCount          int
	EndpointPatterns      []string
}

// DockerfileRequirements defines what must be present in the Dockerfile.
type DockerfileRequirements struct {
	RequireFROM         bool
	RequireEXPOSE       bool
	WarnIfNoUSER        bool
	WarnIfNoHEALTHCHECK bool
}

// CMakeRequirements defines what must be present in CMakeLists.txt.
type CMakeRequirements struct {
	RequireProject       bool
	RequireCXXStandard   bool
	RequireAddExecutable bool
}

// RuntimeAPIContract defines the external surface the bot fleet will call.
type RuntimeAPIContract struct {
	HealthEndpoint   EndpointSpec
	OrderEndpoint    EndpointSpec
	CancelEndpoint   EndpointSpec
	MarketDataStream WebSocketSpec
	RequiredPorts    []int
	EndpointPatterns []string
}

type EndpointSpec struct {
	Required    bool
	Method      string
	Path        string
	ContentType string
	Schema      map[string]string
}

type WebSocketSpec struct {
	Required     bool
	Path         string
	MessageTypes []string
	RequiresPing bool
}
