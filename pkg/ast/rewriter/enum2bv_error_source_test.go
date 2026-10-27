package rewriter

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func enum2BVErrorFixture(t *testing.T) (*ast.ASTManager, *Enum2BVRewriter, *uparams.GlobalParamsService, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	sy, num, b := astTestOwners(t)
	c, err := uparams.NewParamsContext(sy, num, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	astTestMust(t, err)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, err := uparams.NewGlobalParamsService(sy, c, cfg)
	astTestMust(t, err)
	astTestMust(t, g.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(g)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, err := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, err)
		astTestMust(t, g.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, c.Close())
	})
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: num, Budget: b})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.RegisterDeclarationPlugins())
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: astTestSymbol(t, m, "ErrorEnum"), Constructors: []ast.DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "ee0"), Recognizer: astTestSymbol(t, m, "is-ee0")}, {Name: astTestSymbol(t, m, "ee1"), Recognizer: astTestSymbol(t, m, "is-ee1")}}}})
	astTestMust(t, err)
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	_, err = r.utility.IsEnumSort(sorts[0])
	astTestMust(t, err)
	bv, err := m.BitVectorSort(8)
	astTestMust(t, err)
	value, err := m.BitVectorNumeralUnsigned(200, 8)
	astTestMust(t, err)
	fn, err := m.MkFuncDecl(astTestSymbol(t, m, "bad term"), []ast.ASTHandle{bv}, sorts[0], ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	term, err := m.MkApp(fn, []ast.ASTHandle{value})
	astTestMust(t, err)
	astTestPin(t, m, term)
	return m, r, g, sorts[0], term
}

func TestEnum2BVNonFDErrorUsesActualExpressionAndPPParameters(t *testing.T) {
	for _, mode := range []string{"default", "decimal-bv", "negative-bv"} {
		t.Run(mode, func(t *testing.T) {
			m, r, g, _, term := enum2BVErrorFixture(t)
			want := "unable to handle nested data-type expression (|bad term| #xc8)"
			if mode == "decimal-bv" {
				astTestMust(t, g.SourceSetName("pp.bv_literals", "false"))
				want = "unable to handle nested data-type expression (|bad term| (_ bv200 8))"
			}
			if mode == "negative-bv" {
				astTestMust(t, g.SourceSetName("pp.bv_neg", "true"))
				want = "unable to handle nested data-type expression (|bad term| (bvneg #x38))"
			}
			old, err := m.MkTrue()
			astTestMust(t, err)
			out := astTestPin(t, m, old)
			before, err := m.GetRefCount(term)
			astTestMust(t, err)
			_, err = r.reduceArgument(term, &out)
			var base z3exception.Z3Exception
			if err == nil || err.Error() != want || !IsRewriterException(err) || !errors.As(err, &base) || base.HasErrorCode() {
				t.Fatal("source error text/actual printer module mismatch", err, want)
			}
			after, err2 := m.GetRefCount(term)
			astTestMust(t, err2)
			unchanged, err2 := m.RefGet(out)
			astTestMust(t, err2)
			fm, err2 := m.GetFormatManager()
			astTestMust(t, err2)
			if before != after || unchanged != old || fm.Stats().LiveReferences != 0 || len(r.pending) != 0 {
				t.Fatal("error formatting changed borrowed input/output or retained document")
			}
			astTestMust(t, r.Close())
			astTestMust(t, g.Close())
			if err.Error() != want {
				t.Fatal("returned error text borrowed destroyed formatting storage")
			}
		})
	}
}

func TestEnum2BVCheckForFDErrorNamesArgument(t *testing.T) {
	m, r, _, sort, _ := enum2BVErrorFixture(t)
	cs, err := m.DatatypeConstructors(sort)
	astTestMust(t, err)
	value, err := m.MkApp(cs[0], nil)
	astTestMust(t, err)
	astTestPin(t, m, value)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	fn, err := m.MkFuncDecl(astTestSymbol(t, m, "check-nested"), []ast.ASTHandle{sort}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, fn)
	out, proof := enum2BVRefs(t, m)
	_, err = r.config.ReduceApp(fn, ast.SliceArguments([]ast.ASTHandle{value}), out, proof)
	if err == nil || err.Error() != "unable to handle nested data-type expression ee0" || !IsRewriterException(err) {
		t.Fatal("check_for_fd formatted the wrong offending expression", err)
	}
}

func TestEnum2BVErrorPrinterBorrowsInputAndOwnsDocument(t *testing.T) {
	m, _, _, _, term := enum2BVErrorFixture(t)
	before, err := m.GetRefCount(term)
	astTestMust(t, err)
	w := &enum2BVErrorPrinter{manager: m}
	t.Cleanup(func() { astTestMust(t, w.close()) })
	text, err := w.render(term)
	astTestMust(t, err)
	after, err := m.GetRefCount(term)
	astTestMust(t, err)
	if text != "(|bad term| #xc8)" || before != after || w.parameters != nil || w.formatManager.Stats().LiveReferences != 1 {
		t.Fatal("source borrowed expression/owned format_ref lifetime mismatch", text, before, after)
	}
	astTestMust(t, w.close())
	if w.formatManager.Stats().LiveReferences != 0 {
		t.Fatal("error document survived insertion")
	}
}

func TestEnum2BVErrorFormatterRefusalsPropagateWithoutFallback(t *testing.T) {
	t.Run("closed-global", func(t *testing.T) {
		_, r, g, _, term := enum2BVErrorFixture(t)
		astTestMust(t, g.Close())
		err := r.throwNonFD(term)
		if !errors.Is(err, arena.ErrClosed) || IsRewriterException(err) || strings.Contains(err.Error(), "unable to handle") || len(r.pending) != 0 {
			t.Fatal("closed PP owner became a fallback diagnostic", err)
		}
	})
	complete := false
	refusals := 0
	for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, r, _, _, term := enum2BVErrorFixture(t)
			// Warm module registration/format-manager storage, without keeping docs.
			warm := r.throwNonFD(term)
			if warm == nil || !strings.Contains(warm.Error(), "bad term") {
				t.Fatal(warm)
			}
			m.Budget().FailAfter(ordinal)
			err := r.throwNonFD(term)
			m.Budget().FailAfter(-1)
			if errors.Is(err, arena.ErrInjectedFailure) {
				if IsRewriterException(err) {
					t.Fatal("formatter computation refusal became rewriter_exception")
				}
				refusals++
			} else if err != nil && err.Error() == warm.Error() {
				complete = true
			} else {
				t.Fatal("unexpected formatter refusal", ordinal, err)
			}
			fm, err2 := m.GetFormatManager()
			astTestMust(t, err2)
			if fm.Stats().LiveReferences != 0 || len(r.pending) != 0 {
				t.Fatal("formatter refusal retained an owning prefix", ordinal, fm.Stats())
			}
		})
	}
	if !complete || refusals == 0 {
		t.Fatal("formatter reservation census did not complete", complete, refusals)
	}
}

func enum2BVDebugActions(t *testing.T, commands string) (*diagnostic.DebugContext, string) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	astTestMust(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	astTestMust(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "enum-actions-*")
	astTestMust(t, err)
	_, err = inputFile.WriteString(commands)
	astTestMust(t, err)
	_, err = inputFile.Seek(0, 0)
	astTestMust(t, err)
	report, err := os.CreateTemp(t.TempDir(), "enum-report-*")
	astTestMust(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: report, Stderr: report})
	astTestMust(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, inputFile)
	astTestMust(t, err)
	actions, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	astTestMust(t, err)
	astTestMust(t, d.BindAssertionStreams(streams))
	astTestMust(t, d.BindAssertionActions(actions))
	previous, err := diagnostic.BindProcessDebugContext(d)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, err := diagnostic.BindProcessDebugContext(previous)
		astTestMust(t, err)
		astTestMust(t, actions.Close())
		astTestMust(t, input.Close())
		astTestMust(t, streams.Close())
		astTestMust(t, inputFile.Close())
		astTestMust(t, report.Close())
		astTestMust(t, d.Close())
		astTestMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("enum debug owner retained bytes", b.Used())
		}
	})
	return d, report.Name()
}

func TestEnum2BVPopSourceAssertionsAndNativeGate(t *testing.T) {
	t.Run("bounds", func(t *testing.T) {
		m, r := enum2BVBoundFixture(t)
		_, report := enum2BVDebugActions(t, "T")
		err := r.Pop(0)
		if diagnostic.DebugBuild {
			var ex z3exception.DefaultException
			if !errors.As(err, &ex) || ex.Message() != "assertion violation" {
				t.Fatal("bounds SASSERT did not stop Pop", err)
			}
			text, err := os.ReadFile(report)
			astTestMust(t, err)
			if !strings.Contains(string(text), "enum2bv_rewriter.cpp") || !strings.Contains(string(text), "243") || !strings.Contains(string(text), "m_bounds.empty()") {
				t.Fatal("wrong bounds assertion site", string(text))
			}
		} else {
			astTestMust(t, err)
		}
		if len(r.bounds) != 2 || len(r.entries) != 2 {
			t.Fatal("bounds assertion/native Pop0 changed imp vectors")
		}
		var caller []ast.ASTRef
		t.Cleanup(func() {
			for i := range caller {
				astTestMust(t, m.DelRef(&caller[i]))
			}
		})
		astTestMust(t, r.FlushSideConstraints(&caller))
	})
	t.Run("scope-limit", func(t *testing.T) {
		m, _, _ := enum2BVFixture(t)
		r, err := NewEnum2BVRewriter(m)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, r.Close()) })
		astTestMust(t, r.Push())
		_, report := enum2BVDebugActions(t, "T")
		err = r.Pop(2)
		if diagnostic.DebugBuild {
			var ex z3exception.DefaultException
			if !errors.As(err, &ex) || ex.Message() != "assertion violation" {
				t.Fatal("scope limit SASSERT did not stop Pop", err)
			}
			text, err := os.ReadFile(report)
			astTestMust(t, err)
			if !strings.Contains(string(text), "enum2bv_rewriter.cpp") || !strings.Contains(string(text), "245") || !strings.Contains(string(text), "num_scopes <=") {
				t.Fatal("wrong scope limit assertion site", string(text))
			}
		} else if !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("invalid native scope subtraction was not checked", err)
		}
		if len(r.scopeLimits) != 1 {
			t.Fatal("scope assertion changed limits")
		}
	})
}
