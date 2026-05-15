package validator

import (
	"fmt"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	httpMethodSignals = map[string][]string{
		"GET":    {"get", "method_get", "http_get"},
		"POST":   {"post", "method_post", "http_post"},
		"DELETE": {"delete", "del", "method_delete", "http_delete"},
	}
)

// SchemaValidator checks for the existence of an entry point and network listening patterns.
type SchemaValidator struct {
	contract *domain.SubmissionContract
}

func NewSchemaValidator(contract *domain.SubmissionContract) *SchemaValidator {
	return &SchemaValidator{contract: contract}
}

func (v *SchemaValidator) Name() string { return "schema_and_endpoints" }

func (v *SchemaValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
}

func (v *SchemaValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	if !ctx.Main.Exists {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MAIN_UNREADABLE",
			Message:  "Cannot read src/main.cpp",
			Severity: domain.SeverityError,
			FilePath: "src/main.cpp",
		})
		return result
	}

	if !ctx.Main.HasMain {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "NO_MAIN_FUNCTION",
			Message:  "src/main.cpp must contain a standard main() entry point",
			Severity: domain.SeverityError,
			FilePath: "src/main.cpp",
		})
	}

	if !ctx.EndpointSignals.NetworkPattern {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "NO_NETWORK_LISTENER",
			Message:  "Could not detect common network listening patterns (e.g. bind, listen, httplib). Ensure your bot fleet can connect.",
			Severity: domain.SeverityWarning,
			FilePath: "src/main.cpp",
		})
	}

	v.validateEndpoint(ctx, &result, "health", v.contract.RuntimeAPI.HealthEndpoint)
	v.validateEndpoint(ctx, &result, "orders", v.contract.RuntimeAPI.OrderEndpoint)
	v.validateEndpoint(ctx, &result, "cancel", v.contract.RuntimeAPI.CancelEndpoint)
	v.validateWebSocket(ctx, &result, "market_data", v.contract.RuntimeAPI.MarketDataStream)

	return result
}

func (v *SchemaValidator) validateEndpoint(ctx *WorkspaceContext, result *domain.CheckResult, name string, spec domain.EndpointSpec) {
	if spec.Path == "" {
		return
	}

	pathPresent := ctx.EndpointSignals.PathHits[spec.Path]
	methodPresent := methodLikelyPresent(ctx, spec.Method)
	present := pathPresent && methodPresent

	if spec.Required && !present {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_" + strings.ToUpper(name) + "_ENDPOINT",
			Message:  fmt.Sprintf("Required %s endpoint not detected: %s %s", name, spec.Method, spec.Path),
			Severity: domain.SeverityError,
			FilePath: "src/",
		})
		return
	}

	if pathPresent && spec.ContentType != "" && !ctx.EndpointSignals.SchemaFieldHits["content-type"] {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "ENDPOINT_CONTENT_TYPE_UNVERIFIED",
			Message:  fmt.Sprintf("%s endpoint path was detected, but content-type handling could not be verified statically", name),
			Severity: domain.SeverityWarning,
			FilePath: "src/",
		})
	}

	missingFields := missingSchemaFields(ctx, spec.Schema)
	if len(missingFields) > 0 {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "SCHEMA_FIELDS_UNVERIFIED",
			Message:  fmt.Sprintf("%s endpoint schema fields not detected statically: %s", name, strings.Join(missingFields, ", ")),
			Severity: domain.SeverityWarning,
			FilePath: "src/",
		})
	}
}

func (v *SchemaValidator) validateWebSocket(ctx *WorkspaceContext, result *domain.CheckResult, name string, spec domain.WebSocketSpec) {
	if spec.Path == "" {
		return
	}

	pathPresent := ctx.EndpointSignals.WebSocketPathHits[spec.Path]
	present := pathPresent || ctx.EndpointSignals.WebSocketUpgrade
	if spec.Required && !present {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_" + strings.ToUpper(name) + "_WEBSOCKET",
			Message:  fmt.Sprintf("Required WebSocket endpoint not detected: %s", spec.Path),
			Severity: domain.SeverityError,
			FilePath: "src/",
		})
		return
	}

	for _, msgType := range spec.MessageTypes {
		if !ctx.EndpointSignals.WebSocketMessageHits[msgType] {
			result.Warnings = append(result.Warnings, domain.ValidationError{
				Code:     "WEBSOCKET_MESSAGE_UNVERIFIED",
				Message:  fmt.Sprintf("WebSocket message type not detected statically: %s", msgType),
				Severity: domain.SeverityWarning,
				FilePath: "src/",
			})
		}
	}

	if spec.RequiresPing && !ctx.EndpointSignals.PingHandler {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "WEBSOCKET_HEARTBEAT_UNVERIFIED",
			Message:  "WebSocket heartbeat/ping handling was not detected statically",
			Severity: domain.SeverityWarning,
			FilePath: "src/",
		})
	}
}

func methodLikelyPresent(ctx *WorkspaceContext, method string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return true
	}
	if ctx.EndpointSignals.MethodHits[method] {
		return true
	}
	for _, signal := range httpMethodSignals[method] {
		if ctx.EndpointSignals.MethodHits[strings.ToUpper(signal)] {
			return true
		}
	}
	return false
}

func missingSchemaFields(ctx *WorkspaceContext, schema map[string]string) []string {
	if len(schema) == 0 {
		return nil
	}
	missing := make([]string, 0)
	for field := range schema {
		if !ctx.EndpointSignals.SchemaFieldHits[field] {
			missing = append(missing, field)
		}
	}
	return missing
}
