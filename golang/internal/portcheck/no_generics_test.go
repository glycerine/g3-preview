package portcheck

import "testing"

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
	source(t, root, ".cache/ignored.go", "not Go")
	if err := CheckNoGenerics(root); err != nil {
		t.Fatal(err)
	}
}

func TestNoGenericsHiddenSource(t *testing.T) {
	root := t.TempDir()
	source(t, root, ".source/hidden.go", "package p\nfunc F[T any]() {}\n")
	if CheckNoGenerics(root) == nil {
		t.Fatal("hidden project Go source bypassed audit")
	}
}
