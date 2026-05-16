package security

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestSafeArchivePathRejectsTraversal(t *testing.T) {
	cases := []string{"../x", "src/../x", "src/../../x", "/tmp/x", `C:\tmp\x`, "src/\x00/main.cpp"}
	for _, tc := range cases {
		if _, err := SafeArchivePath(tc, ArchiveLimits{}); err == nil {
			t.Fatalf("expected %q to be rejected", tc)
		}
	}
}

func TestValidateZipReaderRejectsDangerousFiles(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(".env")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("SECRET=value")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateZipReader(zr, ArchiveLimits{}); err == nil {
		t.Fatal("expected dangerous file to be rejected")
	}
}

func TestValidateZipReaderAcceptsNormalSubmission(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"src/main.cpp":    "int main() { return 0; }",
		"include/app.hpp": "#pragma once",
		"CMakeLists.txt":  "cmake_minimum_required(VERSION 3.16)",
		"Dockerfile":      "FROM alpine:3.20",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	report, err := ValidateZipReader(zr, ArchiveLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if report.FileCount != len(files) {
		t.Fatalf("file count = %d, want %d", report.FileCount, len(files))
	}
}
