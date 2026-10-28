package portcheck

import (
	"path/filepath"
	"testing"
)

func TestPackageComments(t *testing.T) {
	for _, body := range []string{
		"// Source: src/example.cpp.\npackage sample\n",
		"//go:build linux\n\n// Ported from src/example.cpp.\npackage sample\n",
	} {
		root := t.TempDir()
		source(t, root, "bad.go", body)
		if err := CheckPackageComments(root); err == nil {
			t.Fatalf("pre-package provenance accepted: %q", body)
		}
	}

	root := t.TempDir()
	source(t, root, "doc.go", "// Package sample provides a fixture.\npackage sample\n")
	source(t, root, "source.go", "package sample\n\n// Source: src/example.cpp.\n")
	source(t, root, "tagged.go", "//go:build linux\n\npackage sample\n\n// Source: src/tagged.cpp.\n")
	if err := CheckPackageComments(root); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryPackageComments(t *testing.T) {
	if err := CheckPackageComments(filepath.Join("..", "..", "pkg")); err != nil {
		t.Fatal(err)
	}
}
