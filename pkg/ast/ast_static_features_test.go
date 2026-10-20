package ast

import (
	"bytes"
	"strings"
	"testing"
)

func TestStaticFeaturesSourceResetQueriesAndPrimitiveDisplay(t *testing.T) {
	m := astTestManager(t)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	if !f.CNF || f.MaxStackDepth != 500 || f.HasUF() || f.NumTheories() != 0 {
		t.Fatal("source reset defaults")
	}
	f.NumUninterpretedFunctions = 1
	f.NumMarkedTheories = 2
	if !f.HasUF() || f.NumNonUFTheories() != 2 || f.NumTheories() != 3 {
		t.Fatal("source theory accounting")
	}
	var out bytes.Buffer
	astTestMust(t, f.DisplayPrimitive(&out))
	text := out.String()
	for _, want := range []string{"BEGIN_PRIMITIVE_STATIC_FEATURES\n", "CNF 1\n", "ARITH_K_SUM 0\n", "NUM_THEORIES 3\n", "END_PRIMITIVE_STATIC_FEATURES\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	out.Reset()
	astTestMust(t, f.Display(&out))
	for _, want := range []string{"BEGIN_STATIC_FEATURES\n", "HAS_QUANTIFIERS 0\n", "THEORY_COMBINATION 1\n", "END_STATIC_FEATURES\n"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %q", want, out.String())
		}
	}
	vector := []float64{1, 2}
	if got := f.GetFeatureVector(vector); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatal("empty source feature vector method", got)
	}
	astTestMust(t, f.Reset())
	if !f.CNF || f.NumTheories() != 0 || f.NumUninterpretedFunctions != 0 {
		t.Fatal("reset after populated state")
	}
}
