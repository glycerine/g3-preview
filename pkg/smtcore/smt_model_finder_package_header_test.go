package smtcore

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestSMTModelFinderFilesPutPackageBeforeProvenance(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "smt_model_finder") || !strings.HasSuffix(name, ".go") {
			continue
		}
		file, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		if !scanner.Scan() || scanner.Text() != "package smtcore" {
			_ = file.Close()
			t.Errorf("%s: package statement must precede provenance comments", name)
			continue
		}
		if err := scanner.Err(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
