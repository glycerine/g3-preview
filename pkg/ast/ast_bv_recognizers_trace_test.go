package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTBitVectorAllOneSourceTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	full, err := m.BitVectorNumeralUnsigned(255, 8)
	astTestMust(t, err)
	other, err := m.BitVectorNumeralUnsigned(3, 8)
	astTestMust(t, err)
	if yes, e := m.IsBitVectorAllOne(full); e != nil || !yes || output.Len() != 0 {
		t.Fatalf("disabled trace: %v %v %q", yes, e, output.String())
	}
	tags.Enable("is_allone")
	for _, tc := range []struct {
		handle ASTHandle
		value  bool
	}{{full, true}, {other, false}} {
		got, e := m.IsBitVectorAllOne(tc.handle)
		if e != nil || got != tc.value {
			t.Fatalf("recognizer: got %v, %v; want %v", got, e, tc.value)
		}
	}
	if !tags.ShouldTrace("is_allone") {
		if output.Len() != 0 {
			t.Fatalf("erased trace: %q", output.String())
		}
		return
	}
	want := "-------- [is_allone] is_allone src/ast/bv_decl_plugin.cpp:801 ---------\n255 1\n------------------------------------------------\n" +
		"-------- [is_allone] is_allone src/ast/bv_decl_plugin.cpp:801 ---------\n3 0\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source trace: got %q, want %q", got, want)
	}
}
