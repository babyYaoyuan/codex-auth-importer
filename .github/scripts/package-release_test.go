package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageReleaseCreatesZipAndChecksum(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.WriteFile("codex-auth-importer.so", []byte("plugin"), 0o644); err != nil {
		t.Fatalf("write library: %v", err)
	}
	if err := os.WriteFile("README.md", []byte("# Readme\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile("VERSION", []byte("0.0.0\n"), 0o644); err != nil {
		t.Fatalf("write version: %v", err)
	}

	archive := filepath.Join("dist", "release", "plugin.zip")
	checksum := archive + ".sha256"
	if err := packageRelease("codex-auth-importer.so", archive, checksum); err != nil {
		t.Fatalf("packageRelease() error = %v", err)
	}
	if _, err := os.Stat(checksum); err != nil {
		t.Fatalf("checksum missing: %v", err)
	}
	rawChecksum, err := os.ReadFile(checksum)
	if err != nil {
		t.Fatalf("read checksum: %v", err)
	}
	if !strings.Contains(string(rawChecksum), "plugin.zip") {
		t.Fatalf("checksum = %q, want archive basename", string(rawChecksum))
	}

	zr, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() {
		if err := zr.Close(); err != nil {
			t.Fatalf("close archive: %v", err)
		}
	}()
	names := map[string]bool{}
	for _, file := range zr.File {
		names[file.Name] = true
	}
	for _, want := range []string{"codex-auth-importer.so", "README.md", "VERSION"} {
		if !names[want] {
			t.Fatalf("archive missing %q; got %#v", want, names)
		}
	}
}
