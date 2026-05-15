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
		".cc":    true,
		".cxx":   true,
		".c":     true,
		".h":     true,
		".hh":    true,
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
		RequireFROM:         true,
		RequireEXPOSE:       true,
		WarnIfNoUSER:        true,
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
	RequiredDirs           []string               `json:"required_dirs"`
	RequiredFiles          []string               `json:"required_files"`
	AllowedExtensions      map[string]bool        `json:"allowed_extensions"`
	ForbiddenExtensions    map[string]bool        `json:"forbidden_extensions"`
	ForbiddenPatterns      []string               `json:"forbidden_patterns"`
	DockerfileRequirements DockerfileRequirements `json:"dockerfile_requirements"`
	CMakeRequirements      CMakeRequirements      `json:"cmake_requirements"`
	RuntimeAPI             RuntimeAPIContract     `json:"runtime_api"`
	MaxExtractedBytes      int64                  `json:"max_extracted_bytes"`
	MaxFileCount           int                    `json:"max_file_count"`
	EndpointPatterns       []string               `json:"endpoint_patterns"`
}

// DockerfileRequirements defines what must be present in the Dockerfile.
type DockerfileRequirements struct {
	RequireFROM         bool `json:"require_from"`
	RequireEXPOSE       bool `json:"require_expose"`
	WarnIfNoUSER        bool `json:"warn_if_no_user"`
	WarnIfNoHEALTHCHECK bool `json:"warn_if_no_healthcheck"`
}

// CMakeRequirements defines what must be present in CMakeLists.txt.
type CMakeRequirements struct {
	RequireProject       bool `json:"require_project"`
	RequireCXXStandard   bool `json:"require_cxx_standard"`
	RequireAddExecutable bool `json:"require_add_executable"`
}

// RuntimeAPIContract defines the external surface the bot fleet will call.
type RuntimeAPIContract struct {
	HealthEndpoint   EndpointSpec  `json:"health_endpoint"`
	OrderEndpoint    EndpointSpec  `json:"order_endpoint"`
	CancelEndpoint   EndpointSpec  `json:"cancel_endpoint"`
	MarketDataStream WebSocketSpec `json:"market_data_stream"`
	RequiredPorts    []int         `json:"required_ports"`
	EndpointPatterns []string      `json:"endpoint_patterns"`
}

type EndpointSpec struct {
	Required    bool              `json:"required"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	ContentType string            `json:"content_type,omitempty"`
	Schema      map[string]string `json:"schema,omitempty"`
}

type WebSocketSpec struct {
	Required     bool     `json:"required"`
	Path         string   `json:"path"`
	MessageTypes []string `json:"message_types,omitempty"`
	RequiresPing bool     `json:"requires_ping"`
}
