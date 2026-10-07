// Source: src/ast/datatype_decl_plugin.h:237-246,314-334;
// src/ast/datatype_decl_plugin.cpp:739-750.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTDatatypeActualPluginAndUtilityMembers(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterDatatypePlugin()
	astTestMust(t, e)
	p := m.datatypePlugin
	if p == nil || p.family != fid || !p.published || p.classID != 0 || len(p.definitions.cells) != 8 || p.utility != nil || len(p.definitionBlock) != 0 {
		t.Fatal("actual datatype plugin members", p)
	}
	var u astDatatypeUtility
	astTestMust(t, u.construct(m))
	if u.plugin != p || u.start != 0 || u.completedMaps != 9 || u.references.raw.Block != (util.Block{}) {
		t.Fatal("actual utility completion", u)
	}
	for i := range u.maps {
		if len(u.maps[i].cells) != 8 || u.maps[i].size != 0 {
			t.Fatal("actual reached object-map member", i)
		}
	}
	astTestMust(t, u.destroy())
	astTestMust(t, m.SourceFinalize())
	if len(p.definitions.cells) != 0 || p.header == 0 {
		t.Fatal("completed source plugin DT must release actual definition table before owner history end", len(p.definitions.cells), p.header)
	}
}

func TestASTDatatypeUtilityEveryActualMapConstructorPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal <= 9; ordinal++ {
		m := astTestManager(t)
		_, e := m.RegisterDatatypePlugin()
		astTestMust(t, e)
		b := m.BudgetField
		before := b.Used()
		b.FailAfter(ordinal)
		var u astDatatypeUtility
		call := u.construct(m)
		var actual uint64
		for i := range u.maps {
			actual += uint64(len(u.maps[i].cells)) * 16
			for _, entry := range u.maps[i].cells {
				if entry.key != 0 {
					t.Fatal("default object-map touched payload key", ordinal, i)
				}
			}
		}
		if b.Used() != before+actual {
			t.Fatal("immediate reached utility member graph", ordinal, b.Used(), before, actual)
		}
		astFloatingPointConstructorRawGraph(t, m.numerals, nil)
		if ordinal < 9 {
			if call == nil || u.constructed || actual != 0 || u.completedMaps != uint8(ordinal) || u.bodyCause == nil {
				t.Fatal("completed member reverse unwind", ordinal, call, u.completedMaps, actual)
			}
		} else {
			astTestMust(t, call)
			if actual != 9*8*16 || u.completedMaps != 9 || u.plugin != m.datatypePlugin {
				t.Fatal("literal nine actual arrays", actual, u)
			}
		}
		b.FailAfter(-1)
		if call == nil {
			astTestMust(t, u.destroy())
		}
		if b.Used() != before {
			t.Fatal("utility lexical end", ordinal, b.Used(), before)
		}
	}
}
