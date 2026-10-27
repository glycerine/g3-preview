package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type datatypeUtilTeardownWriter struct {
	action func() error
	err    error
	called bool
}

func (w *datatypeUtilTeardownWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *datatypeUtilTeardownWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		w.err = w.action()
	}
	return len(p), nil
}

func TestDatatypeUtilInterruptedTeardown(t *testing.T) {
	for _, mode := range []string{"reset", "close", "close-after-reset"} {
		t.Run(mode, func(t *testing.T) {
			s, q, b := astTestOwners(t)
			output := &datatypeUtilTeardownWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(output)
			m, err := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, m.Close()) })
			astTestMust(t, m.RegisterDeclarationPlugins())
			sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "teardown-enum"), Constructors: []DatatypeConstructorDefinition{
				{Name: astTestSymbol(t, m, "teardown-a"), Recognizer: astTestSymbol(t, m, "is-teardown-a")},
				{Name: astTestSymbol(t, m, "teardown-b"), Recognizer: astTestSymbol(t, m, "is-teardown-b")},
			}}})
			astTestMust(t, err)
			astTestPin(t, m, sorts[0])
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			constructor, found, err := u.NonRecursiveConstructor(sorts[0])
			astTestMust(t, err)
			if !found {
				t.Fatal("missing constructor")
			}
			allConstructors, err := m.DatatypeConstructors(sorts[0])
			astTestMust(t, err)
			for _, c := range allConstructors {
				astTestPin(t, m, c)
			}
			beforeSort, err := m.GetRefCount(sorts[0])
			astTestMust(t, err)
			beforeConstructor, err := m.GetRefCount(constructor)
			astTestMust(t, err)
			refs := append([]ASTRef(nil), u.refs...)
			start := u.start
			output.action = u.Reset
			if mode == "close" {
				output.action = u.Close
			}
			floatSort, err := m.FloatingPointSort(3, 4)
			astTestMust(t, err)
			tags.Enable("ast")
			if !tags.ShouldTrace("ast") {
				t.Skip("requires source trace build")
			}
			// The numerical constructor's real mutation guard refuses DelRef
			// before consumption. No AST/reference payload is changed by the test.
			_, err = m.MkFPAZeroOfSort(floatSort, false)
			astTestMust(t, err)
			if !output.called || !errors.Is(output.err, ErrASTPending) {
				t.Fatal("missing interrupted retirement", output.called, output.err)
			}
			if !reflect.DeepEqual(u.refs, refs) || u.start != start {
				t.Fatal("interruption consumed references or advanced preference")
			}
			if mode == "close" {
				if u.constructors[sorts[0]] != constructor {
					t.Fatal("close destroyed the map before its reference member")
				}
				if !errors.Is(u.Reset(), ErrASTHandle) {
					t.Fatal("closing utility accepted Reset")
				}
			} else if len(u.constructors) != 0 {
				t.Fatal("reset did not clear the map before reference retirement")
			}
			if _, _, err = u.NonRecursiveConstructor(sorts[0]); !errors.Is(err, ErrASTHandle) {
				t.Fatal("interrupted utility accepted a query", err)
			}
			if mode == "reset" {
				astTestMust(t, u.Reset())
				if u.start != start+1 || u.operation != 0 {
					t.Fatal("reset retry did not advance exactly once")
				}
			} else {
				astTestMust(t, u.Close())
				astTestMust(t, u.Close())
				if u.start != start || u.manager != nil {
					t.Fatal("close replayed Reset or retained manager")
				}
			}
			if len(u.refs) != 0 || len(u.constructors) != 0 {
				t.Fatal("retry retained cached owners")
			}
			afterSort, err := m.GetRefCount(sorts[0])
			astTestMust(t, err)
			afterConstructor, err := m.GetRefCount(constructor)
			astTestMust(t, err)
			if afterSort+2 != beforeSort || afterConstructor+2 != beforeConstructor {
				t.Fatal("retry lost or duplicated a decrement")
			}
			if mode == "reset" {
				_, found, err := u.NonRecursiveConstructor(sorts[0])
				astTestMust(t, err)
				if !found {
					t.Fatal("completed reset did not reopen utility")
				}
			}
		})
	}
}
