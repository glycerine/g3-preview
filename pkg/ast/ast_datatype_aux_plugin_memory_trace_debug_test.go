//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTDatatypeDatalogAndPBPluginMemoryWrappers(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	tags.Enable("memory")
	datatypeID, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	datalogID, err := m.RegisterDatalogPlugin()
	astTestMust(t, err)
	pbID, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	datatype := m.datatypePlugin
	datalog := m.auxiliaryPlugin(astDatalogPlugin)
	pb := m.auxiliaryPlugin(astPseudoBooleanPlugin)
	if datatype == nil || datalog == nil || pb == nil || !datatype.published || !datalog.published || !pb.published {
		t.Fatal("dynamic plugin owners not published")
	}
	sites := []struct {
		identity string
		line     int
		name     string
		extent   uint64
	}{
		{datatype.traceIdentity(), 41, "datatype_decl_plugin", datatype.header},
		{datalog.traceIdentity(), 44, "datalog::dl_decl_plugin", datalog.header},
		{pb.traceIdentity(), 53, "pb_decl_plugin", pb.header},
	}
	for _, site := range sites {
		alloc := "alloc " + site.identity + " src/ast/reg_decl_plugins.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + strconv.FormatUint(site.extent, 10) + "\n"
		if strings.Count(output.String(), alloc) != 1 {
			t.Fatalf("dynamic allocation %q: %q", alloc, output.String())
		}
	}
	if again, e := m.RegisterDatatypePlugin(); e != nil || again != datatypeID || m.datatypePlugin != datatype {
		t.Fatalf("duplicate datatype registration: %d %v", again, e)
	}
	if again, e := m.RegisterDatalogPlugin(); e != nil || again != datalogID || m.auxiliaryPlugin(astDatalogPlugin) != datalog {
		t.Fatalf("duplicate datalog registration: %d %v", again, e)
	}
	if again, e := m.RegisterPseudoBooleanPlugin(); e != nil || again != pbID || m.auxiliaryPlugin(astPseudoBooleanPlugin) != pb {
		t.Fatalf("duplicate PB registration: %d %v", again, e)
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
