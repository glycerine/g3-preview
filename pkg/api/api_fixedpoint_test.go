package api

// Public fixedpoint lifecycle through the retained PDR command integration.

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTContextFixedpointPDRLifecycle(t *testing.T) {
	context, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Close()) })
	manager, err := context.Manager()
	astTestMust(t, err)
	references := make([]ast.ASTRef, 0)
	pin := func(handle ast.ASTHandle, x error) ast.ASTHandle {
		t.Helper()
		astTestMust(t, x)
		reference, x := manager.NewRef(handle)
		astTestMust(t, x)
		references = append(references, reference)
		return handle
	}
	arithmetic, err := manager.ArithmeticFamilyID()
	astTestMust(t, err)
	integer := pin(manager.MkFamilySort(arithmetic, 1, nil))
	zero := pin(manager.ArithmeticIntegerNumeralText("0"))

	astTestMust(t, context.SetFixedpointEngine("pdr"))
	p, err := context.DeclareRelation("api-fp-P", []ast.ASTHandle{integer})
	astTestMust(t, err)
	q, err := context.DeclareRelation("api-fp-Q", []ast.ASTHandle{integer})
	astTestMust(t, err)
	u, err := context.DeclareRelation("api-fp-U", []ast.ASTHandle{integer})
	astTestMust(t, err)
	pZero := pin(manager.MkApp(p, []ast.ASTHandle{zero}))
	uZero := pin(manager.MkApp(u, []ast.ASTHandle{zero}))
	x := pin(context.DeclareFixedpointVariable("api-fp-x", integer))
	pX := pin(manager.MkApp(p, []ast.ASTHandle{x}))
	qX := pin(manager.MkApp(q, []ast.ASTHandle{x}))
	step := pin(manager.MkImplies(pX, qX))
	astTestMust(t, context.PushFixedpoint())
	astTestMust(t, context.AddFixedpointRuleWithBound(uZero, "api-fp-scoped", 3))
	astTestMust(t, context.PopFixedpoint())
	astTestMust(t, context.AddFixedpointRule(pZero, "api-fp-fact"))
	astTestMust(t, context.AddFixedpointRule(step, "api-fp-step"))

	status, err := context.QueryFixedpoint(q)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("reachable public fixedpoint query", status)
	}
	answer, err := context.FixedpointAnswer(q)
	astTestMust(t, err)
	answerHandle, err := manager.RefGet(answer)
	astTestMust(t, err)
	if !manager.IsTrue(answerHandle) {
		t.Fatal("public fixedpoint answer is not true")
	}
	astTestMust(t, manager.DelRef(&answer))
	if unexpected, x := context.FixedpointModel(q); x == nil {
		astTestMust(t, unexpected.Close())
		t.Fatal("reachable fixedpoint query exposed an inductive model")
	}

	levels, err := context.FixedpointNumLevels(q)
	astTestMust(t, err)
	if levels == 0 {
		t.Fatal("public fixedpoint query retained no PDR levels")
	}
	bound := pin(manager.MkVar(0, integer))
	coverProperty := pin(manager.MkEq(bound, zero))
	astTestMust(t, context.AddFixedpointCover(-1, q, coverProperty))
	cover, err := context.FixedpointCoverDelta(-1, q)
	astTestMust(t, err)
	coverHandle, err := manager.RefGet(cover)
	astTestMust(t, err)
	if coverHandle != coverProperty {
		t.Fatalf("public fixedpoint invariant cover = %v, want %v", coverHandle, coverProperty)
	}
	astTestMust(t, manager.DelRef(&cover))
	if err = context.AddFixedpointCover(-2, q, coverProperty); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("invalid public fixedpoint cover level", err)
	}

	status, err = context.QueryFixedpoint(u)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("unreachable public fixedpoint query", status)
	}
	certificate, err := context.FixedpointAnswer(u)
	astTestMust(t, err)
	certificateHandle, err := manager.RefGet(certificate)
	astTestMust(t, err)
	if manager.IsTrue(certificateHandle) {
		t.Fatal("unreachable public fixedpoint query returned an empty certificate")
	}
	astTestMust(t, manager.DelRef(&certificate))
	inductiveModel, err := context.FixedpointModel(u)
	astTestMust(t, err)
	for _, declaration := range []ast.ASTHandle{p, q} {
		present, x := inductiveModel.Core().HasInterpretation(declaration)
		astTestMust(t, x)
		if !present {
			t.Fatal("fixedpoint model omitted retained relation", declaration)
		}
		interpretation, x := inductiveModel.Core().FunctionInterpretation(declaration)
		astTestMust(t, x)
		if interpretation == nil {
			t.Fatal("fixedpoint model omitted relation function", declaration)
		}
		fallback, x := interpretation.Else()
		astTestMust(t, x)
		if fallback.IsNull() {
			t.Fatal("fixedpoint model relation has no interpretation body", declaration)
		}
	}
	if present, x := inductiveModel.Core().HasInterpretation(u); x != nil || present {
		t.Fatal("fixedpoint model retained the unreachable query relation", present, x)
	}
	astTestMust(t, inductiveModel.Close())
	qZero := pin(manager.MkApp(q, []ast.ASTHandle{zero}))
	for _, application := range []ast.ASTHandle{pZero, qZero} {
		value, x := context.EvaluateFixedpointModel(u, application, true)
		astTestMust(t, x)
		handle, x := manager.RefGet(value)
		astTestMust(t, x)
		if !manager.IsTrue(handle) {
			t.Fatal("fixedpoint model does not satisfy reachable relation", application, handle)
		}
		astTestMust(t, manager.DelRef(&value))
	}
	stats, err := context.FixedpointStatistics()
	astTestMust(t, err)
	if stats.Size() == 0 {
		t.Fatal("public fixedpoint statistics are empty")
	}
	astTestMust(t, context.ResetFixedpointStatistics())

	for index := len(references) - 1; index >= 0; index-- {
		astTestMust(t, manager.DelRef(&references[index]))
	}
	astTestMust(t, context.Close())
	if _, err = context.QueryFixedpoint(p); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed public fixedpoint query", err)
	}
}

func TestSMTContextFixedpointParameterUpdateRebuildsPDR(t *testing.T) {
	context, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Close()) })
	astTestMust(t, context.SetFixedpointEngine("pdr"))
	predicate, err := context.DeclareRelation("api-fp-parameter-P", nil)
	astTestMust(t, err)
	manager, err := context.Manager()
	astTestMust(t, err)
	fact, err := manager.MkApp(predicate, nil)
	astTestMust(t, err)
	astTestMust(t, context.AddFixedpointRule(fact, "api-fp-parameter-fact"))
	status, err := context.QueryFixedpoint(predicate)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("initial fixedpoint parameter query", status)
	}
	astTestMust(t, context.SetUIntOption("fixedpoint.pdr.cache_mode", 3))
	if _, err = context.QueryFixedpoint(predicate); err == nil || !strings.Contains(err.Error(), "invalid pdr.cache_mode 3") {
		t.Fatal("fixedpoint parameter update reused stale PDR context", err)
	}
}

func TestSMTContextFixedpointSpacerInductiveExports(t *testing.T) {
	context, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Close()) })
	manager, err := context.Manager()
	astTestMust(t, err)
	astTestMust(t, context.SetFixedpointEngine("spacer"))
	p, err := context.DeclareRelation("api-spacer-P", nil)
	astTestMust(t, err)
	u, err := context.DeclareRelation("api-spacer-U", nil)
	astTestMust(t, err)
	pAtom, err := manager.MkApp(p, nil)
	astTestMust(t, err)
	uAtom, err := manager.MkApp(u, nil)
	astTestMust(t, err)
	astTestMust(t, context.AddFixedpointRule(pAtom, "api-spacer-fact"))
	recursive, err := manager.MkImplies(uAtom, uAtom)
	astTestMust(t, err)
	astTestMust(t, context.AddFixedpointRule(recursive, "api-spacer-recursive"))
	status, err := context.QueryFixedpoint(u)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("Spacer public unreachable query", status)
	}
	certificate, err := context.FixedpointAnswer(u)
	astTestMust(t, err)
	certificateHandle, err := manager.RefGet(certificate)
	astTestMust(t, err)
	if manager.IsTrue(certificateHandle) {
		t.Fatal("Spacer public certificate omitted retained predicates")
	}
	astTestMust(t, manager.DelRef(&certificate))
	inductiveModel, err := context.FixedpointModel(u)
	astTestMust(t, err)
	present, err := inductiveModel.Core().HasInterpretation(p)
	astTestMust(t, err)
	if !present {
		t.Fatal("Spacer public model omitted retained predicate")
	}
	astTestMust(t, inductiveModel.Close())
	value, err := context.EvaluateFixedpointModel(u, pAtom, true)
	astTestMust(t, err)
	valueHandle, err := manager.RefGet(value)
	astTestMust(t, err)
	if !manager.IsTrue(valueHandle) {
		t.Fatal("Spacer public model does not satisfy fact", valueHandle)
	}
	astTestMust(t, manager.DelRef(&value))
	falsity, err := manager.MkFalse()
	astTestMust(t, err)
	astTestMust(t, context.AddFixedpointCover(-1, u, falsity))
	cover, err := context.FixedpointCoverDelta(-1, u)
	astTestMust(t, err)
	coverHandle, err := manager.RefGet(cover)
	astTestMust(t, err)
	if !manager.IsFalse(coverHandle) {
		t.Fatal("Spacer public invariant cover", coverHandle)
	}
	astTestMust(t, manager.DelRef(&cover))
}
