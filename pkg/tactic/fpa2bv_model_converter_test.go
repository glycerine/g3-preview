package tactic

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	astfpa "github.com/glycerine/g3/pkg/ast/fpa"
	"github.com/glycerine/g3/pkg/model"
	modelfpa "github.com/glycerine/g3/pkg/model/fpa"
)

func TestFPA2BVModelConverterRebuildsFloatsAndKeepsOtherDeclarations(t *testing.T) {
	m := astTestManager(t)
	forward, err := astfpa.NewConverter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, forward.Close()) })
	floatSort, err := m.FloatingPointSort(8, 24)
	astTestMust(t, err)
	floatDeclaration, err := m.MkFuncDecl(astTestSymbol(t, m, "model-float"), nil, floatSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	encoded, err := forward.MkConst(floatDeclaration)
	astTestMust(t, err)
	sign, err := m.GetArg(encoded, 0)
	astTestMust(t, err)
	packed, err := m.GetArg(sign, 0)
	astTestMust(t, err)
	packedDeclaration, err := m.GetDecl(packed)
	astTestMust(t, err)
	packedValue, err := m.BitVectorNumeralUnsigned(0x3fc00000, 32)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	plainDeclaration, err := m.MkFuncDecl(astTestSymbol(t, m, "plain-model-value"), nil, boolSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	plainValue, err := m.MkTrue()
	astTestMust(t, err)
	source, err := model.NewModel(m)
	astTestMust(t, err)
	astTestMust(t, source.Core().RegisterConstant(packedDeclaration, packedValue))
	astTestMust(t, source.Core().RegisterConstant(plainDeclaration, plainValue))
	var reference ModelRef
	astTestMust(t, reference.Set(source.Core()))
	t.Cleanup(func() { astTestMust(t, reference.Close()) })
	converter, err := NewFPA2BVModelConverter(m, forward)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, converter.Close()) })
	var display bytes.Buffer
	astTestMust(t, converter.Display(&display))
	if !strings.Contains(display.String(), "fpa2bv-model-converter") || !strings.Contains(display.String(), "model-float") {
		t.Fatal("model converter display", display.String())
	}
	reverse, err := modelfpa.NewConverter(m, forward)
	astTestMust(t, err)
	wantFloat, err := reverse.ConvertPackedBV(nil, floatSort, packedValue)
	astTestMust(t, err)
	_ = astTestPin(t, m, wantFloat)
	astTestMust(t, reverse.Close())
	astTestMust(t, converter.ApplyModel(&reference, 0))
	target, err := reference.Get()
	astTestMust(t, err)
	gotFloat, err := target.ConstantInterpretation(floatDeclaration)
	astTestMust(t, err)
	if gotFloat != wantFloat {
		t.Fatal("rebuilt float", gotFloat, wantFloat)
	}
	gotPlain, err := target.ConstantInterpretation(plainDeclaration)
	astTestMust(t, err)
	if gotPlain != plainValue {
		t.Fatal("plain constant was not preserved", gotPlain, plainValue)
	}
	if consumed, err := target.HasConstantInterpretation(packedDeclaration); err != nil || consumed {
		t.Fatal("auxiliary packed declaration survived", consumed, err)
	}
}
