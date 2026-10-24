package bv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestBitBlasterModelConverterCPPReference(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, boolean := range []bool{true, false} {
		t.Run(fmt.Sprint(boolean), func(t *testing.T) {
			base := sharingTacticFixture(t, false)
			m := base.manager
			binding, bits := blasterBinding(t, base, 8, boolean)
			r := blasterConverter(t, base, []BitBlasterBinding{binding}, boolean)
			tacticName := "bit-blast"
			if !boolean {
				tacticName = "bv1-blast"
			}
			preamble := "(set-logic QF_BV)(set-option :produce-models true)(declare-const original-bv (_ BitVec 8))\n"
			var source, actual strings.Builder
			source.WriteString(preamble)
			for value := 0; value < 256; value++ {
				fmt.Fprintf(&source, "(push 1)(assert (= original-bv #x%02x))(check-sat-using (then %s smt))(get-value (original-bv))(pop 1)\n", value, tacticName)
				md, err := model.NewModel(m)
				sharingMust(t, err)
				var out tactic.ModelRef
				sharingMust(t, out.Set(md.Core()))
				for i, d := range bits {
					position := i
					if !boolean {
						position = 7 - i
					}
					if value&(1<<position) == 0 {
						continue
					}
					var bit ast.ASTHandle
					if boolean {
						bit, err = m.MkTrue()
					} else {
						bit, err = m.BitVectorNumeralUint64(1, 1)
					}
					sharingMust(t, err)
					sharingMust(t, md.Core().RegisterConstant(d, bit))
				}
				sharingMust(t, r.ApplyUnindexed(&out))
				got, err := out.Get()
				sharingMust(t, err)
				number, err := got.ConstantInterpretation(binding.Variable)
				sharingMust(t, err)
				text, _, err := m.BitVectorNumeralText(number)
				sharingMust(t, err)
				u, err := strconv.ParseUint(text, 10, 8)
				sharingMust(t, err)
				fmt.Fprintf(&actual, "sat\n((original-bv #x%02x))\n", u)
				sharingMust(t, out.Close())
			}
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(source.String())
			native, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(native))
			}
			canonical := func(s string) string {
				return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
			}
			if canonical(actual.String()) != canonical(string(native)) {
				t.Fatalf("Go/C++ reconstruction mismatch\nGo: %s\nC++: %s", actual.String(), native)
			}
			cmd = exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(preamble + "(assert (= original-bv #xa5))(apply " + tacticName + " :print false :print_model_converter true)")
			native, err = cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(native))
			}
			fresh := regexp.MustCompile(`k!([0-9]+)`)
			normalized := fresh.ReplaceAllString(string(native), "bit$1")
			var display bytes.Buffer
			sharingMust(t, r.Display(&display))
			if display.String() != normalized {
				t.Fatalf("source converter display\nGo: %s\nC++: %s", display.String(), native)
			}
			for _, width := range []uint32{32, 257} {
				wideBinding, _ := blasterBinding(t, base, width, boolean)
				wide := blasterConverter(t, base, []BitBlasterBinding{wideBinding}, boolean)
				cmd = exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(fmt.Sprintf("(set-logic QF_BV)(set-option :produce-models true)(declare-const original-bv (_ BitVec %d))(assert (= original-bv (_ bv0 %d)))(apply %s :print false :print_model_converter true)", width, width, tacticName))
				native, err = cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(native))
				}
				display.Reset()
				sharingMust(t, wide.Display(&display))
				if display.String() != fresh.ReplaceAllString(string(native), "bit$1") {
					t.Fatalf("source converter display width %d\nGo: %s\nC++: %s", width, display.String(), native)
				}
			}
		})
	}
}

func blasterBinding(t *testing.T, base *MaxBVSharingTactic, width uint32, boolean bool) (BitBlasterBinding, []ast.ASTHandle) {
	t.Helper()
	m := base.manager
	sort, err := m.BitVectorSort(width)
	sharingMust(t, err)
	name, err := m.Symbols.Intern("original-bv")
	sharingMust(t, err)
	x, err := m.MkConst(name, sort)
	sharingMust(t, err)
	xref, err := m.NewRef(x)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, m.DelRef(&xref)) })
	d, err := m.GetDecl(x)
	sharingMust(t, err)
	bitSort, err := m.BitVectorSort(1)
	sharingMust(t, err)
	if boolean {
		bitSort, err = m.MkBoolSort()
		sharingMust(t, err)
	}
	var bits, decls []ast.ASTHandle
	for i := uint32(0); i < width; i++ {
		name, err := m.Symbols.Intern(fmt.Sprint("bit", i))
		sharingMust(t, err)
		b, err := m.MkConst(name, bitSort)
		sharingMust(t, err)
		ref, err := m.NewRef(b)
		sharingMust(t, err)
		t.Cleanup(func() { sharingMust(t, m.DelRef(&ref)) })
		bits = append(bits, b)
		decl, err := m.GetDecl(b)
		sharingMust(t, err)
		decls = append(decls, decl)
	}
	var assembled ast.ASTHandle
	if boolean {
		assembled, err = m.MkBitVectorFromBits(bits)
	} else {
		assembled, err = m.MkBitVectorConcat(bits)
	}
	sharingMust(t, err)
	ref, err := m.NewRef(assembled)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, m.DelRef(&ref)) })
	return BitBlasterBinding{d, assembled}, decls
}

func blasterConverter(t *testing.T, base *MaxBVSharingTactic, bindings []BitBlasterBinding, boolean bool) *BitBlasterModelConverter {
	t.Helper()
	var r *BitBlasterModelConverter
	var err error
	if boolean {
		r, err = NewBitBlasterModelConverter(base.manager, base.context, uparams.BorrowProcessGlobalParamsService(), bindings)
	} else {
		r, err = NewBV1BlasterModelConverter(base.manager, base.context, uparams.BorrowProcessGlobalParamsService(), bindings)
	}
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, r.Close()) })
	return r
}

func TestBitBlasterModelConverterTruthTablesAndWideValues(t *testing.T) {
	for _, boolean := range []bool{true, false} {
		t.Run(fmt.Sprint(boolean), func(t *testing.T) {
			base := sharingTacticFixture(t, false)
			m := base.manager
			binding, bits := blasterBinding(t, base, 8, boolean)
			r := blasterConverter(t, base, []BitBlasterBinding{binding}, boolean)
			for mask := 0; mask < 256; mask++ {
				md, err := model.NewModel(m)
				sharingMust(t, err)
				var out tactic.ModelRef
				sharingMust(t, out.Set(md.Core()))
				for i, d := range bits {
					// Omitted zero bits exercise the source don't-care default.
					if mask&(1<<i) == 0 {
						continue
					}
					var v ast.ASTHandle
					if boolean {
						v, err = m.MkTrue()
					} else {
						v, err = m.BitVectorNumeralUint64(1, 1)
					}
					sharingMust(t, err)
					sharingMust(t, md.Core().RegisterConstant(d, v))
				}
				sharingMust(t, r.ApplyUnindexed(&out))
				got, err := out.Get()
				sharingMust(t, err)
				v, err := got.ConstantInterpretation(binding.Variable)
				sharingMust(t, err)
				text, matched, err := m.BitVectorNumeralText(v)
				sharingMust(t, err)
				want := uint64(mask)
				if !boolean {
					want = 0
					for i := 0; i < 8; i++ {
						want = (want << 1) | uint64(mask>>i&1)
					}
				}
				if !matched || text != fmt.Sprint(want) {
					t.Fatal("bit order", mask, text, want)
				}
				n, err := got.NumConstants()
				sharingMust(t, err)
				if n != 1 {
					t.Fatal("helper bits escaped model", n)
				}
				sharingMust(t, out.Close())
			}
		})
	}
	base := sharingTacticFixture(t, false)
	m := base.manager
	binding, bits := blasterBinding(t, base, 257, true)
	r := blasterConverter(t, base, []BitBlasterBinding{binding}, true)
	md, err := model.NewModel(m)
	sharingMust(t, err)
	var out tactic.ModelRef
	sharingMust(t, out.Set(md.Core()))
	defer func() { sharingMust(t, out.Close()) }()
	yes, err := m.MkTrue()
	sharingMust(t, err)
	sharingMust(t, md.Core().RegisterConstant(bits[0], yes))
	sharingMust(t, md.Core().RegisterConstant(bits[256], yes))
	sharingMust(t, r.ApplyUnindexed(&out))
	got, err := out.Get()
	sharingMust(t, err)
	v, err := got.ConstantInterpretation(binding.Variable)
	sharingMust(t, err)
	text, _, err := m.BitVectorNumeralText(v)
	sharingMust(t, err)
	want := new(big.Int).Lsh(big.NewInt(1), 256)
	want.Add(want, big.NewInt(1))
	if text != want.String() {
		t.Fatal("wide reconstruction truncated", text, want)
	}
}

func TestBitBlasterModelConverterNoncanonicalAndEmptyModels(t *testing.T) {
	for _, boolean := range []bool{true, false} {
		t.Run(fmt.Sprint(boolean), func(t *testing.T) {
			base := sharingTacticFixture(t, false)
			m := base.manager
			binding, bits := blasterBinding(t, base, 4, boolean)
			r := blasterConverter(t, base, []BitBlasterBinding{binding}, boolean)
			md, err := model.NewModel(m)
			sharingMust(t, err)
			var out tactic.ModelRef
			sharingMust(t, out.Set(md.Core()))
			defer func() { sharingMust(t, out.Close()) }()
			var zero, one, sort ast.ASTHandle
			if boolean {
				zero, err = m.MkFalse()
				sharingMust(t, err)
				one, err = m.MkTrue()
				sharingMust(t, err)
				sort, err = m.MkBoolSort()
			} else {
				zero, err = m.BitVectorNumeralUint64(0, 1)
				sharingMust(t, err)
				one, err = m.BitVectorNumeralUint64(1, 1)
				sharingMust(t, err)
				sort, err = m.BitVectorSort(1)
			}
			sharingMust(t, err)
			name, err := m.Symbols.Intern("nonliteral-bit-value")
			sharingMust(t, err)
			other, err := m.MkConst(name, sort)
			sharingMust(t, err)
			ref, err := m.NewRef(other)
			sharingMust(t, err)
			defer func() { sharingMust(t, m.DelRef(&ref)) }()
			for i, value := range []ast.ASTHandle{zero, one, other} {
				sharingMust(t, md.Core().RegisterConstant(bits[i], value))
			}
			sharingMust(t, r.ApplyUnindexed(&out))
			got, err := out.Get()
			sharingMust(t, err)
			value, err := got.ConstantInterpretation(binding.Variable)
			sharingMust(t, err)
			text, _, err := m.BitVectorNumeralText(value)
			sharingMust(t, err)
			want := "2" // Boolean: only literal true contributes, LSB first.
			if !boolean {
				want = "6"
			} // BV1: every present nonzero term contributes, MSB first.
			if text != want {
				t.Fatal("source noncanonical-value rule", text, want)
			}
			empty := blasterConverter(t, base, nil, boolean)
			before := got
			sharingMust(t, empty.ApplyUnindexed(&out))
			after, err := out.Get()
			sharingMust(t, err)
			if before == after {
				t.Fatal("empty converter did not create a new model")
			}
			value, err = after.ConstantInterpretation(binding.Variable)
			sharingMust(t, err)
			text, _, err = m.BitVectorNumeralText(value)
			sharingMust(t, err)
			if text != want {
				t.Fatal("empty converter lost a non-bit constant", text)
			}
		})
	}
}

func TestBitBlasterModelConverterPreservesModelAndTranslation(t *testing.T) {
	base := sharingTacticFixture(t, false)
	m := base.manager
	binding, bits := blasterBinding(t, base, 4, true)
	r := blasterConverter(t, base, []BitBlasterBinding{binding}, true)
	var display bytes.Buffer
	sharingMust(t, r.Display(&display))
	if !strings.Contains(display.String(), "(bit-blaster-model-converter\n  (original-bv (mkbv bit0 bit1 bit2 bit3))") {
		t.Fatal(display.String())
	}
	if err := r.Display(nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	md, err := model.NewModel(m)
	sharingMust(t, err)
	var old, out tactic.ModelRef
	sharingMust(t, old.Set(md.Core()))
	sharingMust(t, out.Set(md.Core()))
	defer func() { sharingMust(t, out.Close()); sharingMust(t, old.Close()) }()
	value, err := m.BitVectorNumeralUint64(11, 4)
	sharingMust(t, err)
	sharingMust(t, md.Core().RegisterConstant(binding.Variable, value))
	yes, err := m.MkTrue()
	sharingMust(t, err)
	sharingMust(t, md.Core().RegisterConstant(bits[0], yes))
	name, err := m.Symbols.Intern("universe")
	sharingMust(t, err)
	sort, err := m.MkUninterpretedSort(name, nil)
	sharingMust(t, err)
	u, err := m.MkModelValue(0, sort)
	sharingMust(t, err)
	sharingMust(t, md.RegisterUniverse(sort, []ast.ASTHandle{u}))
	name, err = m.Symbols.Intern("uninterpreted-function")
	sharingMust(t, err)
	f, err := m.MkFuncDecl(name, []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	sharingMust(t, err)
	fi, err := model.NewFunctionInterpretation(m, 1)
	sharingMust(t, err)
	sharingMust(t, fi.SetElse(u))
	sharingMust(t, md.Core().RegisterFunction(f, fi))
	var trace bytes.Buffer
	r.trace = diagnostic.NewTraceTagsWithWriter(&trace)
	r.trace.Enable("blaster_mc")
	sharingMust(t, r.ApplyUnindexed(&out))
	got, err := out.Get()
	sharingMust(t, err)
	actual, err := got.ConstantInterpretation(binding.Variable)
	sharingMust(t, err)
	if actual != value || got == md.Core() {
		t.Fatal("existing interpretation or independent model", actual, value)
	}
	if r.trace.ShouldTrace("blaster_mc") && (!strings.Contains(trace.String(), "collect_bits src/tactic/bv/bit_blaster_model_converter.cpp:67") || !strings.Contains(trace.String(), "non-bit: original-bv\n")) {
		t.Fatal(trace.String())
	}
	copyFI, err := got.FunctionInterpretation(f)
	sharingMust(t, err)
	if copyFI == fi || copyFI == nil {
		t.Fatal("function graph ownership was shared")
	}
	universe, err := got.DerivedModel().Universe(sort)
	sharingMust(t, err)
	if len(universe) != 1 || universe[0] != u {
		t.Fatal("universe was lost", universe)
	}
	sharingMust(t, old.Close())
	elseValue, err := copyFI.Else()
	sharingMust(t, err)
	if elseValue != u {
		t.Fatal("copied graph was not independent")
	}
	destination, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	sharingMust(t, err)
	defer func() { sharingMust(t, destination.Close()) }()
	tr, err := ast.NewASTTranslation(m, destination, true)
	sharingMust(t, err)
	defer func() { sharingMust(t, tr.Close()) }()
	other, err := r.Translate(tr)
	sharingMust(t, err)
	translated := other.(*BitBlasterModelConverter)
	defer func() { sharingMust(t, translated.Close()) }()
	var translatedDisplay bytes.Buffer
	sharingMust(t, translated.Display(&translatedDisplay))
	if translatedDisplay.String() != display.String() {
		t.Fatal("translated display", translatedDisplay.String())
	}
	if ast.ASTSourceDebug {
		if err := r.ApplyModel(&out, 1); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("goal index", err)
		}
	}
	copy := *r
	if err := copy.Close(); !errors.Is(err, tactic.ErrModelConverterOwner) {
		t.Fatal("copied owner", err)
	}
	sharingMust(t, r.IncRef())
	sharingMust(t, r.DecRef())
	if err := r.Cancel(); !errors.Is(err, tactic.ErrModelConverterOwner) {
		t.Fatal("closed owner", err)
	}
}

func TestBitBlasterModelConverterAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "translate"} {
		success := false
		for stop := int64(0); stop < 512; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				base := sharingTacticFixture(t, false)
				m := base.manager
				binding, bits := blasterBinding(t, base, 4, true)
				r := blasterConverter(t, base, []BitBlasterBinding{binding}, true)
				var run func() error
				if phase == "constructor" {
					run = func() error {
						fresh, err := NewBitBlasterModelConverter(m, r.context, r.globals, []BitBlasterBinding{binding})
						if fresh != nil {
							err = errors.Join(err, fresh.Close())
						}
						return err
					}
				}
				if phase == "apply" {
					md, err := model.NewModel(m)
					sharingMust(t, err)
					var out tactic.ModelRef
					sharingMust(t, out.Set(md.Core()))
					defer func() { sharingMust(t, out.Close()) }()
					yes, err := m.MkTrue()
					sharingMust(t, err)
					sharingMust(t, md.Core().RegisterConstant(bits[3], yes))
					run = func() error { return r.ApplyUnindexed(&out) }
				}
				if phase == "translate" {
					destination, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
					sharingMust(t, err)
					defer func() { sharingMust(t, destination.Close()) }()
					tr, err := ast.NewASTTranslation(m, destination, true)
					sharingMust(t, err)
					defer func() { sharingMust(t, tr.Close()) }()
					run = func() error {
						fresh, err := r.Translate(tr)
						if fresh != nil {
							err = errors.Join(err, fresh.Close())
						}
						return err
					}
				}
				m.Budget().FailAfter(stop)
				err := run()
				m.Budget().FailAfter(-1)
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if success {
				break
			}
		}
		if !success {
			t.Fatal("no successful reservation frontier", phase)
		}
	}
}
