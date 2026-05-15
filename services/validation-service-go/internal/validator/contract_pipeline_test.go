package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iicpc/validation-service-go/internal/domain"
)

func TestPipelinePassesCompleteSubmissionContract(t *testing.T) {
	root := writeContractWorkspace(t, "8080", `
#include <httplib.h>
int main() {
  httplib::Server server;
  server.Get("/health", [](auto&, auto&){});
  server.Post("/api/v1/orders", [](auto& req, auto&){
    auto body = "id symbol side price quantity type application/json";
  });
  server.Delete("/api/v1/orders/:id", [](auto&, auto&){});
  auto ws = "/ws/market-data book_snapshot trade heartbeat ping pong";
  server.listen("0.0.0.0", 8080);
  return 0;
}
`)

	report := NewPipeline(&domain.DefaultContract).Run("sub-1", root)
	if report.Status != domain.ValidationPassed {
		t.Fatalf("expected validation to pass, got %s: %#v", report.Status, report.CheckResults)
	}
	if report.Compatibility == nil || !report.Compatibility.Compatible {
		t.Fatalf("expected compatible report, got %#v", report.Compatibility)
	}
	if report.ExposedPort != 8080 {
		t.Fatalf("expected exposed port 8080, got %d", report.ExposedPort)
	}
}

func TestPipelineRejectsMissingRequiredPortAndAPISurface(t *testing.T) {
	root := writeContractWorkspace(t, "9090", `
int main() {
  return 0;
}
`)

	report := NewPipeline(&domain.DefaultContract).Run("sub-2", root)
	if report.Status != domain.ValidationFailed {
		t.Fatalf("expected validation to fail")
	}
	if report.Compatibility == nil || report.Compatibility.Compatible {
		t.Fatalf("expected incompatible report, got %#v", report.Compatibility)
	}

	schema := report.CheckResults["schema_and_endpoints"]
	if schema.Passed {
		t.Fatalf("expected schema_and_endpoints to fail")
	}
	ports := report.CheckResults["port_binding"]
	if ports.Passed {
		t.Fatalf("expected port_binding to fail")
	}
}

func writeContractWorkspace(t *testing.T, exposedPort string, mainCPP string) string {
	t.Helper()

	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src"))
	mustMkdir(t, filepath.Join(root, "include"))
	mustWrite(t, filepath.Join(root, "src", "main.cpp"), mainCPP)
	mustWrite(t, filepath.Join(root, "include", "engine.hpp"), "#pragma once\n")
	mustWrite(t, filepath.Join(root, "CMakeLists.txt"), `
cmake_minimum_required(VERSION 3.20)
project(contract_test)
set(CMAKE_CXX_STANDARD 20)
add_executable(contract_test src/main.cpp)
`)
	mustWrite(t, filepath.Join(root, "Dockerfile"), "FROM alpine:3.20\nEXPOSE "+exposedPort+"\nUSER 10001\nHEALTHCHECK CMD wget -qO- http://localhost:"+exposedPort+"/health || exit 1\n")
	return root
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
