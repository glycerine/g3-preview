package tactic

import (
	"bytes"
	"io"
	"math"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type refusingExtensionTraceWriter struct{ calls int }

func (w *refusingExtensionTraceWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrClosedPipe
}

func TestExtensionModelTraceWriterRefusalKeepsDefinitions(t *testing.T) {
	w := &refusingExtensionTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("extension_mc")
	m, md, out, config := extensionFixture(t, tags)
	c, err := NewExtensionModel(m, config)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	a := goalTestAtom(t, m, "extension-refused-trace")
	d, err := m.GetDecl(a)
	satOK(t, err)
	yes, err := m.MkTrue()
	satOK(t, err)
	satOK(t, c.Insert(d, yes))
	satOK(t, c.ApplyModel(out, 0))
	got, err := md.Core().ConstantInterpretation(d)
	satOK(t, err)
	if got != yes {
		t.Fatal("source extension definition stopped after trace stream refusal", got)
	}
	if tags.ShouldTrace("extension_mc") && w.calls != 1 {
		t.Fatal("source extension trace stream did not retain bad state", w.calls)
	}
}

// Source: extension_model_converter.cpp:57-68. The source traverses the
// definitions backward and prints each evaluated value before publishing it.
func TestExtensionModelDefinitionSourceTraceReverseOrder(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, _, out, config := extensionFixture(t, tags)
	c, err := NewExtensionModel(m, config)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	a := goalTestAtom(t, m, "extension-trace-a")
	b := goalTestAtom(t, m, "extension-trace-b")
	da, err := m.GetDecl(a)
	satOK(t, err)
	db, err := m.GetDecl(b)
	satOK(t, err)
	yes, err := m.MkTrue()
	satOK(t, err)
	satOK(t, c.Insert(da, b))
	satOK(t, c.Insert(db, yes))
	satOK(t, c.ApplyModel(out, 0))
	if output.Len() != 0 {
		t.Fatal("disabled extension trace", output.String())
	}
	tags.Enable("extension_mc")
	satOK(t, c.ApplyModel(out, 0))
	if !tags.ShouldTrace("extension_mc") {
		if output.Len() != 0 {
			t.Fatal("erased extension trace", output.String())
		}
		return
	}
	idA, err := m.GetASTID(da)
	satOK(t, err)
	idB, err := m.GetASTID(db)
	satOK(t, err)
	modelBody := "extension-trace-b -> true\n" +
		"extension-trace-a -> true\n" +
		"extension-trace-b (Bool)  :id " + strconv.FormatUint(uint64(idB), 10) + "\n" +
		"extension-trace-a (Bool)  :id " + strconv.FormatUint(uint64(idA), 10) + "\n"
	modelRecord := func(line string) string {
		return "-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:" + line + " ---------\n" +
			modelBody + "------------------------------------------------\n"
	}
	record := func(name, definition string) string {
		return "-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:59 ---------\n" +
			name + " ->\n" + definition + "\n==>\ntrue\n" +
			"------------------------------------------------\n"
	}
	want := modelRecord("49") + record("extension-trace-b", "true") + record("extension-trace-a", "extension-trace-b") + modelRecord("71")
	if got := output.String(); got != want {
		t.Fatalf("source extension definition records: got %q, want %q", got, want)
	}
}

// Source: extension_model_converter.cpp:29-44,49,71. The declaration-info
// suffix is emitted after the reconstructed model's V2 constant row.
func TestExtensionModelTraceAttachedDeclarationInfo(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, _, out, config := extensionFixture(t, tags)
	c, err := NewExtensionModel(m, config)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	bs, err := m.MkBoolSort()
	satOK(t, err)
	p, err := m.NewParameterInt(7)
	satOK(t, err)
	defer func() { satOK(t, m.DelParameter(&p)) }()
	negativeZero, err := m.NewParameterDouble(math.Copysign(0, -1))
	satOK(t, err)
	defer func() { satOK(t, m.DelParameter(&negativeZero)) }()
	exponent, err := m.NewParameterDouble(1_000_000)
	satOK(t, err)
	defer func() { satOK(t, m.DelParameter(&exponent)) }()
	info, err := m.NewFuncDeclInfo(ast.NullFamilyID, 17, []ast.ASTParameter{p, negativeZero, exponent}, ast.ASTFuncFlags{Skolem: true}, false)
	satOK(t, err)
	defer func() { satOK(t, m.DelFuncDeclInfo(&info)) }()
	d, err := m.MkFuncDecl(astTestSymbol(t, m, "metadata-extension"), nil, bs, info)
	satOK(t, err)
	yes, err := m.MkTrue()
	satOK(t, err)
	satOK(t, c.Insert(d, yes))
	tags.Enable("extension_mc")
	satOK(t, c.ApplyModel(out, 0))
	if !tags.ShouldTrace("extension_mc") {
		if output.Len() != 0 {
			t.Fatal("erased extension metadata trace", output.String())
		}
		return
	}
	id, err := m.GetASTID(d)
	satOK(t, err)
	meta := ":fid -1 :decl-kind 17 :parameters (7 -0 1e+06)" +
		" :left-assoc 0 :right-assoc 0 :flat-associative 0 :commutative 0" +
		" :chainable 0 :pairwise 0 :injective 0 :idempotent 0 :skolem 1"
	want := "-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:49 ---------\n" +
		"------------------------------------------------\n" +
		"-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:59 ---------\n" +
		"metadata-extension ->\ntrue\n==>\ntrue\n" +
		"------------------------------------------------\n" +
		"-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:71 ---------\n" +
		"metadata-extension -> true\n" +
		"metadata-extension (Bool) " + meta + " :id " + strconv.FormatUint(uint64(id), 10) + "\n" +
		"------------------------------------------------\n"
	if output.String() != want {
		t.Fatalf("source extension metadata trace: got %q, want %q", output.String(), want)
	}
}
