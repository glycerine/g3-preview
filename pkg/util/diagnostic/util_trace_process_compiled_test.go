//go:build g3trace

package diagnostic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessTraceFileOpensBeforeFirstEmission(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".z3-trace")
	if err := os.WriteFile(path, []byte("old run\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tags := newProcessTraceTagsAt(path)
	if tags.openErr != nil {
		t.Fatal(tags.openErr)
	}
	file, ok := tags.writer.(*os.File)
	if !ok {
		t.Fatal("process trace did not own its startup file")
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if got, err := os.ReadFile(path); err != nil || len(got) != 0 {
		t.Fatal("startup open must truncate before any enabled tag", string(got), err)
	}
	tags.Enable("test")
	if err := tags.Emit("test", "record", "source.cpp", 7, "body\n"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "-------- [test] record source.cpp:7 ---------\nbody\n------------------------------------------------\n" {
		t.Fatal("source process trace file output", string(got), err)
	}
	tags.Finalize()
	if _, err := file.Stat(); err != nil {
		t.Fatal("finalize_trace must leave the process stream open", err)
	}
	failed := newProcessTraceTagsAt(filepath.Join(t.TempDir(), "missing", ".z3-trace"))
	if failed.openErr == nil || failed.writer != nil {
		t.Fatal("failed startup file open must retain the source failure state")
	}
	failed.Enable("test")
	if err := failed.Emit("test", "record", "source.cpp", 7, "body\n"); err != failed.openErr {
		t.Fatal("failed startup stream must not retry on emission", err)
	}
}
