package validator

import (
	"github.com/iicpc/validation-service-go/internal/domain"
)

// LanguageDetector infers the primary language and runtime of the submission.
// All file analysis is done once by AnalyzeWorkspace; this detector only
// inspects the pre-computed WorkspaceContext.
type LanguageDetector struct{}

func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{}
}

func (v *LanguageDetector) Name() string { return "language_detection" }

func (v *LanguageDetector) DetectContext(ctx *WorkspaceContext) domain.LanguageInfo {
	info := domain.LanguageInfo{
		Language: "unknown",
		Runtime:  "unknown",
		Standard: "unknown",
	}

	if ctx.CMake.Exists {
		info.Language = "cpp"
		if ctx.CMake.StandardVersion != "" {
			info.Standard = ctx.CMake.StandardVersion
			info.Runtime = "C++" + ctx.CMake.StandardVersion
		} else {
			info.Runtime = "C++"
		}
		return info
	}

	counts := ctx.ExtensionCounts
	if counts[".cpp"] > 0 || counts[".cc"] > 0 || counts[".cxx"] > 0 {
		info.Language = "cpp"
		info.Runtime = "C++"
	} else if counts[".c"] > 0 {
		info.Language = "c"
		info.Runtime = "C"
	} else if counts[".rs"] > 0 {
		info.Language = "rust"
		info.Runtime = "Rust"
	} else if counts[".go"] > 0 {
		info.Language = "go"
		info.Runtime = "Go"
	}

	return info
}
