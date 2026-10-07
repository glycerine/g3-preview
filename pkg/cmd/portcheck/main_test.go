package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/portcheck"
)

func TestRunRepositoryLedgerAuditsPkgModule(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg", "util")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "pkg", "go.mod"), "module fixture\n")
	impl := filepath.Join(dir, "impl.go")
	write(impl, "package util\nfunc One() {}\n")
	write(filepath.Join(dir, "impl_test.go"), "package util\nimport \"testing\"\nfunc TestOne(t *testing.T) {}\n")
	manifest := filepath.Join(root, "port_manifest.json")
	m := portcheck.Manifest{Schema: portcheck.SchemaVersion, Units: []portcheck.Unit{{
		Path: "src/util/fixture.cpp", Status: "partial",
		GoTargets: []string{"pkg/util/impl.go"}, Tests: []string{"TestOne"},
	}}}
	if err := portcheck.Save(manifest, m); err != nil {
		t.Fatal(err)
	}
	if err := run("", manifest, false); err != nil {
		t.Fatal("repository-relative targets", err)
	}
	write(impl, "package util\nfunc One[T any]() {}\n")
	if err := run("", manifest, false); err == nil || !strings.Contains(err.Error(), "Go generics prohibited") {
		t.Fatal("root-level ledger must still audit its nested pkg module", err)
	}
}
