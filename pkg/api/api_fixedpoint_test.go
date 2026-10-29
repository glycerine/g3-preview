package api

// Public fixedpoint lifecycle through the retained PDR command integration.

import (
	"errors"
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

	astTestMust(t, context.SetFixedpointEngine("spacer"))
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
