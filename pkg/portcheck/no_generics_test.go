package portcheck

import (
	"path/filepath"
	"testing"
)

func TestNoGenericsAudit(t *testing.T) {
	for _, body := range []string{
		"package p\ntype Box[T any] struct { Value T }\n",
		"package p\nfunc Copy[T any](x T) T { return x }\n",
		"package p\nimport \"reflect\"\nvar t = reflect.TypeFor[uint32]()\n",
		"package p\nvar x = Lookup[uint32,string]()\n",
		"package p\ntype Number interface { ~uint32 }\n",
	} {
		root := t.TempDir()
		source(t, root, "bad.go", body)
		if err := CheckNoGenerics(root); err == nil {
			t.Fatalf("generic Go accepted: %s", body)
		}
	}
	root := t.TempDir()
	source(t, root, "ok.go", "package p\ntype Box struct { Value uint32 }\nfunc At(xs []uint32, i int) uint32 { return xs[i] }\n")
	source(t, root, "dict.go", "package p\nimport ordered \"github.com/glycerine/insdict\"\nvar d ordered.Dict[uint32,string]\n")
	source(t, root, ".cache/ignored.go", "not Go")
	if err := CheckNoGenerics(root); err != nil {
		t.Fatal(err)
	}
}

func TestNoGenericsRejectsUnapprovedExternalInstantiation(t *testing.T) {
	root := t.TempDir()
	source(t, root, "bad.go", "package p\nimport external \"example.com/generic\"\nvar d external.Dict[uint32,string]\n")
	if err := CheckNoGenerics(root); err == nil {
		t.Fatal("generic from unapproved dependency bypassed audit")
	}
}

func TestNoGenericsHiddenSource(t *testing.T) {
	root := t.TempDir()
	source(t, root, ".source/hidden.go", "package p\nfunc F[T any]() {}\n")
	if CheckNoGenerics(root) == nil {
		t.Fatal("hidden project Go source bypassed audit")
	}
}

func TestNoGenericsAuditSeparateModuleBoundary(t *testing.T) {
	root := t.TempDir()
	source(t, root, "go.mod", "module mainmodule\n")
	source(t, root, "ok.go", "package p\n")
	source(t, root, "worktrees/candidate/go.mod", "module candidate\n")
	source(t, root, "worktrees/candidate/bad.go", "package p\nfunc F[T any]() {}\n")
	if err := CheckNoGenerics(root); err != nil {
		t.Fatalf("separate candidate module leaked into main audit: %v", err)
	}
	if err := CheckNoGenerics(filepath.Join(root, "worktrees/candidate")); err == nil {
		t.Fatal("candidate's direct audit accepted generics")
	}
	source(t, root, "worktrees/production/bad.go", "package p\nfunc F[T any]() {}\n")
	if err := CheckNoGenerics(root); err == nil {
		t.Fatal("ordinary project source inside worktrees bypassed audit")
	}
}
