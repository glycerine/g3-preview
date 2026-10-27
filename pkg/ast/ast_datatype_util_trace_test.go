package ast

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func datatypeTraceFixture(t *testing.T, out io.Writer) (*ASTManager, *DatatypeUtil, *diagnostic.TraceTags) {
	t.Helper()
	c, g, _, budget := dynAckSourceFixture(t)
	previous, err := uparams.BindProcessGlobalParamsService(g)
	astTestMust(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, err) })
	tags := diagnostic.NewTraceTagsWithWriter(out)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	return m, u, tags
}

func datatypeTraceBlock(line int, body string) string {
	function := "get_non_rec_constructor_core"
	if line == 914 {
		function = "get_non_rec_constructor"
	}
	return fmt.Sprintf("-------- [util_bug] %s src/ast/datatype_decl_plugin.cpp:%d ---------\n%s------------------------------------------------\n", function, line, body)
}

func TestDatatypeUtilExactSourceTraceAndWarmCache(t *testing.T) {
	var output bytes.Buffer
	m, u, tags := datatypeTraceFixture(t, &output)
	sort := datatypeSelectionEnum(t, m)
	tags.Enable("util_bug")
	chosen, found, err := u.NonRecursiveConstructor(sort)
	astTestMust(t, err)
	if !found {
		t.Fatal("missing enum preference")
	}
	want := ""
	if tags.ShouldTrace("util_bug") {
		want = datatypeTraceBlock(914, "invoke get-non-rec: selection-enum\n") +
			datatypeTraceBlock(937, "get-non-rec constructor: selection-enum\nforbidden: selection-enum \nconstructors: 3\n(declare-fun selection-a () selection-enum)\n(declare-fun selection-b () selection-enum)\n(declare-fun selection-c () selection-enum)\n") +
			datatypeTraceBlock(948, "checking selection-enum: (declare-fun selection-b () selection-enum)\n") +
			datatypeTraceBlock(953, "found non-rec (declare-fun selection-b () selection-enum)\n")
	}
	if output.String() != want {
		t.Fatalf("source trace mismatch\ngot %q\nwant %q", output.String(), want)
	}
	before := m.Stats().LiveReferences
	other, found, err := u.NonRecursiveConstructor(sort)
	astTestMust(t, err)
	if !found || other != chosen || output.String() != want || m.Stats().LiveReferences != before || len(u.tracePending) != 0 {
		t.Fatal("warm preference hit traced, reacquired references or changed selection")
	}
	format, err := m.GetFormatManager()
	astTestMust(t, err)
	if format.Stats().LiveReferences != 0 {
		t.Fatal("trace retained a format_ref")
	}
}

type datatypeRefusingTraceWriter struct {
	bytes.Buffer
	once    func()
	refuse  bool
	calls   int
	observe func([]byte)
	at      string
}

func (w *datatypeRefusingTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *datatypeRefusingTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.observe != nil {
		w.observe(p)
	}
	if w.once != nil && (w.at == "" || strings.HasSuffix(w.String()+string(p), w.at)) {
		f := w.once
		w.once = nil
		f()
	}
	if w.refuse {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(p)
}

func TestDatatypeUtilTraceTemporaryReferenceLifetimes(t *testing.T) {
	w := &datatypeRefusingTraceWriter{}
	m, u, tags := datatypeTraceFixture(t, w)
	sort := datatypeSelectionEnum(t, m)
	constructors, err := u.Constructors(sort)
	astTestMust(t, err)
	format, err := m.GetFormatManager()
	astTestMust(t, err)
	before := m.Stats().LiveReferences
	sortCount, err := m.GetRefCount(sort)
	astTestMust(t, err)
	declCount, err := m.GetRefCount(constructors[1])
	astTestMust(t, err)
	seen, wrong := false, false
	w.observe = func(p []byte) {
		if seen || !strings.HasPrefix(string(p), "(declare-fun selection-b") || !strings.Contains(w.String(), "checking selection-enum: ") {
			return
		}
		seen = true
		sc, e1 := m.GetRefCount(sort)
		dc, e2 := m.GetRefCount(constructors[1])
		wrong = e1 != nil || e2 != nil || sc != sortCount+1 || dc != declCount+1 || m.Stats().LiveReferences != before+2 || format.Stats().LiveReferences != 1
	}
	tags.Enable("util_bug")
	_, found, err := u.NonRecursiveConstructor(sort)
	astTestMust(t, err)
	if !found || tags.ShouldTrace("util_bug") && (!seen || wrong) {
		t.Fatal("sort_ref/func_decl_ref temporaries or format_ref lost their source lifetime", seen, wrong)
	}
	if m.Stats().LiveReferences != before+2 || format.Stats().LiveReferences != 0 {
		t.Fatal("trace retained temporary reference owners")
	}
}

func TestDatatypeUtilTraceHeaderFailurePrecedesCounter(t *testing.T) {
	w := &datatypeRefusingTraceWriter{at: "get-non-rec constructor: "}
	m, u, tags := datatypeTraceFixture(t, w)
	sort := datatypeSelectionEnum(t, m)
	_, err := u.Constructors(sort)
	astTestMust(t, err)
	before := m.Stats().LiveReferences
	w.once = func() { m.Budget().FailAfter(0) }
	tags.Enable("util_bug")
	_, found, err := u.NonRecursiveConstructor(sort)
	m.Budget().FailAfter(-1)
	if !tags.ShouldTrace("util_bug") {
		astTestMust(t, err)
		if !found {
			t.Fatal("erased header refused selection")
		}
		return
	}
	if !errors.Is(err, arena.ErrInjectedFailure) || found || u.start != 0 || len(u.refs) != 4 || len(u.constructors) != 0 || m.Stats().LiveReferences != before || !strings.HasSuffix(w.String(), "get-non-rec constructor: ") {
		t.Fatal("core header computation advanced the counter or changed the acquired cache prefix", found, err, u.start, w.String())
	}
}

func TestDatatypeUtilTraceStickyStreamStillEvaluatesOperands(t *testing.T) {
	w := &datatypeRefusingTraceWriter{refuse: true}
	m, u, tags := datatypeTraceFixture(t, w)
	sort := datatypeSelectionEnum(t, m)
	tags.Enable("util_bug")
	_, found, err := u.NonRecursiveConstructor(sort)
	astTestMust(t, err)
	if !found || u.start != 1 || len(u.refs) != 6 || len(u.tracePending) != 0 {
		t.Fatal("ostream failure changed selection or retained temporaries")
	}
	if tags.ShouldTrace("util_bug") && w.calls != 1 {
		t.Fatal("sticky ostream performed later physical writes", w.calls)
	}
	format, err := m.GetFormatManager()
	astTestMust(t, err)
	if tags.ShouldTrace("util_bug") && format.formatPlugin == nil {
		t.Fatal("failed ostream skipped actual sort/decl formatting")
	}
	if format.Stats().LiveReferences != 0 {
		t.Fatal("failed ostream retained a format owner")
	}
}

func TestDatatypeUtilTraceComputationFailureStopsBeforeCounterAndCache(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		t.Run(fmt.Sprint(refuse), func(t *testing.T) {
			w := &datatypeRefusingTraceWriter{refuse: refuse}
			m, u, tags := datatypeTraceFixture(t, w)
			sort := datatypeSelectionEnum(t, m)
			before := m.Stats().LiveReferences
			w.once = func() { m.Budget().FailAfter(0) }
			tags.Enable("util_bug")
			_, found, err := u.NonRecursiveConstructor(sort)
			m.Budget().FailAfter(-1)
			if tags.ShouldTrace("util_bug") {
				if !errors.Is(err, arena.ErrInjectedFailure) || found || u.start != 0 || len(u.refs) != 0 || len(u.lists) != 0 || len(u.constructors) != 0 || len(u.tracePending) != 0 || m.Stats().LiveReferences != before {
					t.Fatal("trace computation failure lost its source prefix or leaked owners", found, err, u.start)
				}
				if !refuse && (!strings.HasSuffix(w.String(), "invoke get-non-rec: ") || strings.Contains(w.String(), "------------------------------------------------")) {
					t.Fatal("failed formatter lost prefix or emitted footer", w.String())
				}
			} else {
				astTestMust(t, err)
				if !found || w.calls != 0 {
					t.Fatal("erased trace evaluated its body")
				}
			}
		})
	}
}

func TestDatatypeUtilRecursiveSourceTracePreservesForbiddenOrder(t *testing.T) {
	var output bytes.Buffer
	m, u, tags := datatypeTraceFixture(t, &output)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	constructor := func(name string, fields ...DatatypeAccessorDefinition) DatatypeConstructorDefinition {
		return DatatypeConstructorDefinition{Name: astTestSymbol(t, m, name), Recognizer: astTestSymbol(t, m, "is-"+name), Accessors: fields}
	}
	field := func(name string, index uint32) DatatypeAccessorDefinition {
		return DatatypeAccessorDefinition{Name: astTestSymbol(t, m, name), Recursive: true, RecursiveIndex: index}
	}
	sorts, err := m.MkDatatypes([]DatatypeDefinition{
		{Name: astTestSymbol(t, m, "TA"), Constructors: []DatatypeConstructorDefinition{constructor("a", DatatypeAccessorDefinition{Name: astTestSymbol(t, m, "flag"), Range: boolean}, field("a-tail", 1))}},
		{Name: astTestSymbol(t, m, "TB"), Constructors: []DatatypeConstructorDefinition{constructor("b0", field("b-back", 0)), constructor("b1", field("b-tail", 2))}},
		{Name: astTestSymbol(t, m, "TC"), Constructors: []DatatypeConstructorDefinition{constructor("c0", field("c-back", 0)), constructor("leaf")}},
	})
	astTestMust(t, err)
	tags.Enable("util_bug")
	chosen, found, err := u.NonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	if !found || u.start != 3 || len(u.lists) != 3 || len(u.constructors) != 1 || len(u.refs) != 10 || len(u.tracePending) != 0 {
		t.Fatal("trace changed recursive search or leaked temporary owners")
	}
	name, err := m.GetASTName(chosen)
	astTestMust(t, err)
	text, err := m.Symbols.String(name)
	astTestMust(t, err)
	if text != "a" {
		t.Fatal("recursive source preference", text)
	}
	if !tags.ShouldTrace("util_bug") {
		if output.Len() != 0 {
			t.Fatal("erased recursive trace emitted output")
		}
		return
	}
	last := -1
	for _, part := range []string{"non_rec_constructor c: 0 (declare-fun a", "Bool is not a datatype\n", "get-non-rec constructor: TB\n", "TA is in forbidden_set\n", "get-non-rec constructor: TC\n", "nested_c: leaf\n", "nested_c: b1\n"} {
		position := strings.Index(output.String(), part)
		if position <= last {
			t.Fatalf("missing or reordered source trace %q in\n%s", part, output.String())
		}
		last = position
	}
	if !strings.Contains(output.String(), "forbidden: TA TB TC \n") {
		t.Fatal("forbidden vector lost insertion order", output.String())
	}
}

func TestDatatypeUtilTraceConsultsActualPPGlobalOwner(t *testing.T) {
	var output bytes.Buffer
	m, u, tags := datatypeTraceFixture(t, &output)
	sort := datatypeSelectionEnum(t, m)
	before := m.Stats().LiveReferences
	astTestMust(t, uparams.BorrowProcessGlobalParamsService().Close())
	tags.Enable("util_bug")
	_, found, err := u.NonRecursiveConstructor(sort)
	if tags.ShouldTrace("util_bug") {
		if !errors.Is(err, arena.ErrClosed) || found || u.start != 0 || len(u.refs) != 0 || m.Stats().LiveReferences != before || !strings.HasSuffix(output.String(), "invoke get-non-rec: ") {
			t.Fatal("trace substituted defaults for its closed PP global owner", found, err, output.String())
		}
	} else {
		astTestMust(t, err)
		if !found || output.Len() != 0 {
			t.Fatal("erased trace reached PP ownership")
		}
	}
}
