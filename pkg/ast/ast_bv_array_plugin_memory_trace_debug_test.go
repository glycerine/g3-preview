//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTBitVectorAndArrayPluginMemoryWrappersSourceOrder(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	tags.Enable("memory")
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	bvID, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	arrayID, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	if m.bitVectorPlugin == nil || m.arrayPlugin == nil || !m.bitVectorPlugin.published || !m.arrayPlugin.published {
		t.Fatal("dynamic plugin owners not published")
	}
	bv, array := m.bitVectorPlugin, m.arrayPlugin
	sites := []struct {
		identity string
		line     int
		name     string
		extent   uint64
	}{
		{fmt.Sprintf("%p", m.arithmeticPlugin), 32, "arith_decl_plugin", m.arithmeticPlugin.header},
		{bv.traceIdentity(), 35, "bv_decl_plugin", bv.header},
		{array.traceIdentity(), 38, "array_decl_plugin", array.header},
	}
	for _, site := range sites {
		alloc := "alloc " + site.identity + " src/ast/reg_decl_plugins.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + strconv.FormatUint(site.extent, 10) + "\n"
		if strings.Count(output.String(), alloc) != 1 {
			t.Fatalf("dynamic allocation %q: %q", alloc, output.String())
		}
	}
	if again, e := m.RegisterBitVectorPlugin(); e != nil || again != bvID || m.bitVectorPlugin != bv {
		t.Fatalf("duplicate bit-vector registration: %d %v", again, e)
	}
	if again, e := m.RegisterArrayPlugin(); e != nil || again != arrayID || m.arrayPlugin != array {
		t.Fatalf("duplicate array registration: %d %v", again, e)
	}
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	got := output.String()
	previous := -1
	for _, site := range sites {
		alloc := "alloc " + site.identity + " src/ast/reg_decl_plugins.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + strconv.FormatUint(site.extent, 10) + "\n"
		dealloc := "dealloc " + site.identity + " src/ast/ast.cpp:1412\n"
		ai, di := strings.Index(got, alloc), strings.Index(got, dealloc)
		if ai < 0 || di <= ai || di <= previous || strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 {
			t.Fatalf("dynamic plugin lifetime/order for %q: %q", site.identity, got)
		}
		previous = di
	}
}
