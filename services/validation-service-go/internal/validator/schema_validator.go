package validator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	mainFunctionRegex = regexp.MustCompile(`(?m)^int\s+main\s*\(`)
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
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	mainPath := filepath.Join(rootDir, "src", "main.cpp")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MAIN_UNREADABLE",
			Message:  "Cannot read src/main.cpp: " + err.Error(),
			Severity: domain.SeverityError,
			FilePath: "src/main.cpp",
		})
		return result
	}

	contentStr := string(content)

	if !mainFunctionRegex.MatchString(contentStr) && !strings.Contains(contentStr, "int main(") {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "NO_MAIN_FUNCTION",
			Message:  "src/main.cpp must contain a standard main() entry point",
			Severity: domain.SeverityError,
			FilePath: "src/main.cpp",
		})
	}

	// Warn if no endpoint patterns are found (soft check, as they might abstract it heavily).
	foundEndpointPattern := false
	for _, pattern := range v.contract.EndpointPatterns {
		if strings.Contains(contentStr, pattern) {
			foundEndpointPattern = true
			break
		}
	}

	if !foundEndpointPattern {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "NO_NETWORK_LISTENER",
			Message:  "Could not detect common network listening patterns (e.g. bind, listen, httplib). Ensure your bot fleet can connect.",
			Severity: domain.SeverityWarning,
			FilePath: "src/main.cpp",
		})
	}

	return result
}
