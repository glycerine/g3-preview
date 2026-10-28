package smtcore

// Source: src/smt/theory_fpa.cpp SASSERT gates, including lines 495-498.

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSMTFPASourceAssertionGateAndSortShape(t *testing.T) {
	d, report := newLRASourceAssertionContext(t)
	astTestMust(t, d.EnableAssertions(false))
	evaluated := false
	if err := fpaSourceAssert(498, "n->get_owner()->get_decl()->get_range() == s", func() (bool, error) {
		evaluated = true
		return false, nil
	}); err != nil || evaluated {
		t.Fatal("disabled FPA SASSERT evaluated its condition", evaluated, err)
	}

	var trace strings.Builder
	manager, theory, _, fpSort, _ := smtFPAInternalizeTraceFixture(t, &trace)
	constant, err := manager.MkConst(astTestSymbol(t, manager, "fpa-assertion-sort-owner"), fpSort)
	astTestMust(t, err)
	state, err := theory.State()
	astTestMust(t, err)
	node, err := state.MkEnode(constant, false, false, true)
	astTestMust(t, err)
	roundingSort, err := manager.RoundingModeSort()
	astTestMust(t, err)
	if err = theory.ApplySortConstraint(node, roundingSort); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("disabled source assertion changed the physical sort guard", err)
	}

	astTestMust(t, d.EnableAssertions(true))
	err = theory.ApplySortConstraint(node, roundingSort)
	if diagnostic.DebugBuild {
		var assertion z3exception.DefaultException
		if !errors.As(err, &assertion) || assertion.Message() != "assertion violation" {
			t.Fatal("enabled FPA source assertion did not reject a mismatched range", err)
		}
	} else if !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("release FPA sort guard changed", err)
	}
	reportText, readErr := os.ReadFile(report)
	astTestMust(t, readErr)
	if diagnostic.DebugBuild {
		if !strings.Contains(string(reportText), "File: src/smt/theory_fpa.cpp\nLine: 498\nn->get_owner()->get_decl()->get_range() == s\n") {
			t.Fatal("FPA source assertion report", string(reportText))
		}
	} else if len(reportText) != 0 {
		t.Fatal("release FPA SASSERT was not erased", string(reportText))
	}
}
