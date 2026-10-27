package ast

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestDatatypeConstructorCacheOwnsAllDeclarationsIndependently(t *testing.T) {
	m := astTestManager(t)
	sort := datatypeSelectionEnum(t, m)
	constructors, err := m.DatatypeConstructors(sort)
	astTestMust(t, err)
	handles := append([]ASTHandle{sort}, constructors...)
	before := make([]uint32, len(handles))
	for i, h := range handles {
		astTestPin(t, m, h)
		before[i], err = m.GetRefCount(h)
		astTestMust(t, err)
	}
	a, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, a.Close()) })
	b, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	for _, u := range []*DatatypeUtil{a, b} {
		values, err := u.Constructors(sort)
		astTestMust(t, err)
		if len(values) != 3 || len(u.refs) != 4 || len(u.constructors) != 0 || u.start != 0 {
			t.Fatal("constructor cache acquired preference owners")
		}
		values[0] = ASTHandle{}
		again, err := u.Constructors(sort)
		astTestMust(t, err)
		if again[0] != constructors[0] || len(u.refs) != 4 {
			t.Fatal("snapshot mutation corrupted cached vector or cache hit reacquired pins")
		}
	}
	if a.lists[sort] == b.lists[sort] {
		t.Fatal("independent utilities share a vector owner")
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i]+2 {
			t.Fatal("unselected constructor missing independent cache pins", i, count, before[i])
		}
	}
	astTestMust(t, a.Reset())
	if len(a.lists) != 0 || len(a.refs) != 0 || a.start != 1 || len(b.lists) != 1 {
		t.Fatal("Reset did not isolate its cache owner")
	}
	astTestMust(t, b.Close())
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i] {
			t.Fatal("cache teardown lost a caller pin or retained an owner", i, count, before[i])
		}
	}
	if _, err := b.Constructors(sort); !errors.Is(err, ErrASTHandle) {
		t.Fatal("closed constructor cache revived", err)
	}
}

func TestDatatypeConstructorCacheSourceForwardDeletionOrder(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort := datatypeSelectionEnum(t, m)
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	_, found, err := u.NonRecursiveConstructor(sort)
	astTestMust(t, err)
	if !found {
		t.Fatal("missing preference")
	}
	constructors, err := u.Constructors(sort)
	astTestMust(t, err)
	ids := make([]uint32, len(constructors))
	for i, h := range constructors {
		ids[i], err = m.GetASTID(h)
		astTestMust(t, err)
	}
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		t.Skip("requires source trace build")
	}
	astTestMust(t, u.Close())
	// m_asts owns sort, c0, c1, c2, sort, selected c1. The first
	// c1 decrement leaves its preference pin alive, so deletion is 0,2,1.
	last := -1
	for _, i := range []int{0, 2, 1} {
		position := strings.Index(output.String(), fmt.Sprintf("Deleting object %d ", ids[i]))
		if position <= last {
			t.Fatal("cache/reference-vector source deletion order changed", i, output.String())
		}
		last = position
		if _, err := m.GetKind(constructors[i]); err == nil {
			t.Fatal("cache leaked constructor", i)
		}
	}
	if _, err := m.GetKind(sort); err != nil {
		t.Fatal("cache consumed the independent caller sort pin", err)
	}
}

type datatypePluginFinalizationWriter struct {
	bytes.Buffer
	plugin           *astDatatypePluginOwner
	target           string
	seen, wrongOrder bool
}

func (w *datatypePluginFinalizationWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *datatypePluginFinalizationWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if !w.seen && w.target != "" && strings.Contains(w.String(), w.target) {
		w.seen = true
		p := w.plugin
		w.wrongOrder = len(p.definitionRefs) != 0 || p.definitions != nil || p.definitionNames != nil || p.utility == nil || p.membersEnded
	}
	return n, err
}

func TestASTDatatypePluginOwnsAndFinalizesConcreteUtility(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	output := &datatypePluginFinalizationWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "plugin-cache"), Constructors: []DatatypeConstructorDefinition{
		{Name: astTestSymbol(t, m, "plugin-cache-a"), Recognizer: astTestSymbol(t, m, "is-plugin-cache-a")},
		{Name: astTestSymbol(t, m, "plugin-cache-b"), Recognizer: astTestSymbol(t, m, "is-plugin-cache-b")},
		{Name: astTestSymbol(t, m, "plugin-cache-c"), Recognizer: astTestSymbol(t, m, "is-plugin-cache-c")},
	}}})
	astTestMust(t, err)
	p := m.datatypePlugin
	if p.utility != nil {
		t.Fatal("plugin utility was not lazy")
	}
	chosen, found, err := m.DatatypeNonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	u := p.utility
	if !found || u == nil || len(u.refs) != 6 || len(u.lists[sorts[0]].values) != 3 || u.constructors[sorts[0]] != chosen {
		t.Fatal("plugin bypassed the concrete cached utility")
	}
	_, found, err = m.DatatypeNonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	if !found || p.utility != u || len(u.refs) != 6 || u.start != 1 {
		t.Fatal("plugin cache hit reconstructed its utility")
	}
	id, err := m.GetASTID(u.lists[sorts[0]].values[0])
	astTestMust(t, err)
	output.plugin = p
	output.target = fmt.Sprintf("Deleting object %d ", id)
	tags.Enable("ast")
	// Close enters the actual plugin finalizer after definition retirement;
	// its captured utility must be closed before manager node collection.
	astTestMust(t, m.Close())
	if !p.membersEnded || p.utility != nil || u.manager != nil || len(u.refs) != 0 || len(u.lists) != 0 || m.liveReferences != 0 {
		t.Fatal("plugin finalization bypassed utility reference retirement")
	}
	if tags.ShouldTrace("ast") && (!output.seen || output.wrongOrder) {
		t.Fatal("definition map outlived entry into utility reference destruction", output.seen, output.wrongOrder)
	}
}

func TestDatatypeConstructorCacheRetainsStoppedPublishedPrefixes(t *testing.T) {
	for prefix := uint32(0); prefix <= 5; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			constructors, err := m.DatatypeConstructors(sort)
			astTestMust(t, err)
			for _, h := range constructors {
				astTestPin(t, m, h)
			}
			var pins []ASTRef
			t.Cleanup(func() {
				for i := range pins {
					astTestMust(t, m.DelRef(&pins[i]))
				}
			})
			// Place the next actual reference-page reservation at sort pin,
			// constructor-local acquisition, or one of the three cache pins.
			for m.freeReference != 0 || (m.nextReference-1+prefix)%astPageSize != 0 {
				ref, err := m.NewRef(sort)
				astTestMust(t, err)
				pins = append(pins, ref)
			}
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			before := m.Stats().LiveReferences
			m.Budget().FailAfter(0)
			values, err := u.Constructors(sort)
			m.Budget().FailAfter(-1)
			if prefix == 5 {
				astTestMust(t, err)
				if len(values) != 3 || len(u.refs) != 4 {
					t.Fatal("complete source cache missing pins")
				}
			} else {
				if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal("missing actual cache allocation refusal", err)
				}
				wantRefs, wantValues := 0, 0
				if prefix > 0 {
					wantRefs = 1
				}
				if prefix > 2 {
					wantValues = int(prefix) - 2
					wantRefs += wantValues
				}
				if len(u.refs) != wantRefs || u.lexical != (ASTRef{}) || len(u.constructors) != 0 || u.start != 0 {
					t.Fatal("stopped constructor cache lost owners or leaked lexical ref", len(u.refs), wantRefs)
				}
				if prefix == 0 {
					if len(u.lists) != 0 {
						t.Fatal("cache published before sort pin")
					}
				} else {
					cached, err := u.Constructors(sort)
					astTestMust(t, err)
					if len(cached) != wantValues || len(u.refs) != wantRefs {
						t.Fatal("warm lookup refilled the stopped source prefix")
					}
					for i, h := range cached {
						if h != constructors[i] {
							t.Fatal("stopped cache lost declaration order")
						}
					}
				}
			}
			astTestMust(t, u.Close())
			if m.Stats().LiveReferences != before {
				t.Fatal("cache close retained an acquired reference record")
			}
		})
	}
}
