package portcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func source(t *testing.T, root, path, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// Added inventory coverage: tests use temporary synthetic trees, never ../src.
func TestInventoryCompleteDeterministic(t *testing.T) {
	root := t.TempDir()
	source(t, root, "src/qe/qe.cpp", "#ifdef ACTIVE\nSASSERT(x);\n#endif\n")
	source(t, root, "src/api/z3_interp.h", "// Z3_API Z3_fake();\nint Z3_API Z3_compute_interpolant(int x);\n")
	source(t, root, "src/test/qe.cpp", "void tst_qe() { ENSURE(true); }\n")
	source(t, root, "src/qe/CMakeLists.txt", "z3_add_component(qe SOURCES qe.cpp)\n")
	source(t, root, "src/api/python/z3.py", "# binding reference\n")
	source(t, root, "scripts/mk_project.py", "add_lib('qe', ['ast'])\n")
	a, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || len(a.Units) != 6 {
		t.Fatalf("unstable or missing inventory: %+v", a)
	}
	if len(a.APIs) != 1 || a.APIs[0].Name != "Z3_compute_interpolant" {
		t.Fatalf("API inventory: %+v", a.APIs)
	}
	if a.Units[3].Path != "src/qe/CMakeLists.txt" {
		t.Fatalf("not sorted: %+v", a.Units)
	}
	var qe *Unit
	for i := range a.Units {
		if a.Units[i].Path == "src/qe/qe.cpp" {
			qe = &a.Units[i]
		}
	}
	if qe == nil || qe.Component != "qe" || qe.Status != "unported" || len(qe.Conditionals) != 2 || len(qe.Assertions) != 1 {
		t.Fatalf("missing metadata: %+v", qe)
	}
	if err := Validate(a); err != nil {
		t.Fatal(err)
	}
	if err := Compare(a, b); err != nil {
		t.Fatal(err)
	}
	source(t, root, "src/qe/new.cpp", "// new\n")
	c, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := Compare(a, c); err == nil {
		t.Fatal("missing new source accepted")
	}
}

func TestDriftAndCompletionEvidence(t *testing.T) {
	root := t.TempDir()
	source(t, root, "src/api/z3_api.h", "void Z3_API Z3_one();\n")
	a, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	source(t, root, "src/api/z3_api.h", "void Z3_API Z3_two();\n")
	b, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if Compare(a, b) == nil {
		t.Fatal("source/API drift accepted")
	}
	a.Units[0].Status = "ported"
	if Validate(a) == nil {
		t.Fatal("ported without method/test/ownership audit accepted")
	}
	a.Units[0].GoTargets = []string{"api.go"}
	a.Units[0].Tests = []string{"TestOne"}
	a.Units[0].MethodsAudited = true
	a.Units[0].OwnershipAudited = true
	a.Units[0].Methods = []Method{{Name: "Z3_one", Status: "ported", GoTarget: "api.go", Tests: []string{"TestOne"}}}
	if err := Validate(a); err != nil {
		t.Fatal(err)
	}
	a.Units[0].Methods[0].Status = "unported"
	if Validate(a) == nil {
		t.Fatal("unfinished method hidden by file status")
	}
}

func TestMalformedLedger(t *testing.T) {
	for _, m := range []Manifest{
		{Schema: SchemaVersion, Units: []Unit{{Path: "../escape", Status: "unported"}}},
		{Schema: SchemaVersion, Units: []Unit{{Path: "src/a.cpp", Status: "unported"}, {Path: "src/a.cpp", Status: "unported"}}},
		{Schema: SchemaVersion, Units: []Unit{{Path: "src/a.cpp", Status: "done"}}},
	} {
		if Validate(m) == nil {
			t.Fatalf("accepted malformed manifest: %+v", m)
		}
	}
}

func TestExecutableEvidence(t *testing.T) {
	root := t.TempDir()
	source(t, root, "api.go", "package example\nfunc One() {}\n")
	source(t, root, "api_test.go", "package example\nimport \"testing\"\nfunc TestOne(t *testing.T) {}\n")
	m := Manifest{Schema: SchemaVersion, Units: []Unit{{Path: "src/api/one.cpp", Status: "partial", GoTargets: []string{"api.go"}, Tests: []string{"TestOne"}}}}
	if err := VerifyEvidence(m, root); err != nil {
		t.Fatal(err)
	}
	m.Units[0].Tests = []string{"TestMissing"}
	if VerifyEvidence(m, root) == nil {
		t.Fatal("nonexistent test accepted")
	}
	m.Units[0].Tests = []string{"TestOne"}
	m.Units[0].GoTargets = []string{"missing.go"}
	if VerifyEvidence(m, root) == nil {
		t.Fatal("nonexistent implementation accepted")
	}
}

// The reference source is fixed; inventory stores semantic coverage, never
// source-content fingerprints or hashes.
func TestInventoryDoesNotStoreSourceContentHashes(t *testing.T) {
	root := t.TempDir()
	source(t, root, "src/qe/qe.cpp", "#ifdef ACTIVE\nSASSERT(x);\n#endif\n")
	m, e := Inventory(root)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(root, "manifest.json")
	if e = Save(path, m); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]json.RawMessage
	if e = json.Unmarshal(b, &doc); e != nil {
		t.Fatal(e)
	}
	if _, ok := doc["source_fingerprint"]; ok {
		t.Error("stored source fingerprint")
	}
	var units []map[string]json.RawMessage
	if e = json.Unmarshal(doc["units"], &units); e != nil {
		t.Fatal(e)
	}
	for _, u := range units {
		if _, ok := u["sha256"]; ok {
			t.Error("stored source content hash")
		}
	}
}
func TestCompareIgnoresSourceContentChanges(t *testing.T) {
	root := t.TempDir()
	source(t, root, "src/qe/qe.cpp", "#ifdef ACTIVE\nint value = 1;\n#endif\n")
	a, e := Inventory(root)
	if e != nil {
		t.Fatal(e)
	}
	source(t, root, "src/qe/qe.cpp", "#ifdef ACTIVE\nint value = 2; // changed implementation, same inventory\n#endif\n")
	b, e := Inventory(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = Compare(a, b); e != nil {
		t.Fatal("content-only changes must not cause hash drift", e)
	}
	if len(b.Units) != 1 || len(b.Units[0].Conditionals) != 2 {
		t.Fatal("conditional inventory lost", b)
	}
	source(t, root, "src/qe/extra.cpp", "// additional source unit\n")
	c, e := Inventory(root)
	if e != nil {
		t.Fatal(e)
	}
	if Compare(a, c) == nil {
		t.Fatal("new unit must remain detected")
	}
}
