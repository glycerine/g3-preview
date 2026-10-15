//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTRegisterDeclarationPluginsMemoryWrapperSourceOrder(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	tags.Enable("memory")
	astTestMust(t, m.RegisterDeclarationPlugins())
	arith, bv, array, datatype := m.arithmeticPlugin, m.bitVectorPlugin, m.arrayPlugin, m.datatypePlugin
	datalog, seq, fpa, pb := m.auxiliaryPlugin(astDatalogPlugin), m.sequencePlugin, m.floatingPointPlugin, m.auxiliaryPlugin(astPseudoBooleanPlugin)
	if arith == nil || bv == nil || array == nil || datatype == nil || datalog == nil || seq == nil || fpa == nil || pb == nil {
		t.Fatal("registration did not publish all eight source plugin owners")
	}
	sites := []struct {
		identity string
		line     int
		name     string
		extent   uint64
	}{
		{arith.traceIdentity(), 32, "arith_decl_plugin", arith.header},
		{bv.traceIdentity(), 35, "bv_decl_plugin", bv.header},
		{array.traceIdentity(), 38, "array_decl_plugin", array.header},
		{datatype.traceIdentity(), 41, "datatype_decl_plugin", datatype.header},
		{datalog.traceIdentity(), 44, "datalog::dl_decl_plugin", datalog.header},
		{seq.traceIdentity(), 47, "seq_decl_plugin", seq.header},
		{fpa.traceIdentity(), 50, "fpa_decl_plugin", fpa.header},
		{pb.traceIdentity(), 53, "pb_decl_plugin", pb.header},
	}
	allocations := output.String()
	previous := -1
	for _, site := range sites {
		alloc := "alloc " + site.identity + " src/ast/reg_decl_plugins.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + strconv.FormatUint(site.extent, 10) + "\n"
		index := strings.Index(allocations, alloc)
		if index <= previous || strings.Count(allocations, alloc) != 1 {
			t.Fatalf("registration allocation order for %q: %q", site.identity, allocations)
		}
		previous = index
	}
	astTestMust(t, m.RegisterDeclarationPlugins())
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	got := output.String()
	previous = -1
	for _, site := range sites {
		alloc := "alloc " + site.identity + " src/ast/reg_decl_plugins.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + strconv.FormatUint(site.extent, 10) + "\n"
		dealloc := "dealloc " + site.identity + " src/ast/ast.cpp:1412\n"
		ai, di := strings.Index(got, alloc), strings.Index(got, dealloc)
		if ai < 0 || di <= ai || di <= previous || strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 {
			t.Fatalf("registration lifetime/order for %q: %q", site.identity, got)
		}
		previous = di
	}
}
