package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTPBDisplayExactNativeWatchPhasesAndScopeRollback(t *testing.T) {
	_, s, pb, atom, ls := smtPBWatchFixture(t, []int64{1, 1}, 2, false)
	var out bytes.Buffer
	astTestMust(t, s.DisplayTheories(&out))
	want := "watch (v): p2 |-> p1 \nwatch (v): p3 |-> p1 \np1@(l_undef) ((_ pbge 2 1 1) native-pb-0 native-pb-1)\np2@(l_undef) + p3@(l_undef) >= 2\nsum: [0:2] \n"
	if out.String() != want {
		t.Fatal("initial source variable watches", out.String(), want)
	}
	astTestMust(t, s.PushScope())
	smtPBWatchAssign(t, s, atom)
	out.Reset()
	astTestMust(t, pb.Display(&out))
	want = "watch: p2 |-> p1 \nwatch: p3 |-> p1 \nwatch (v): p2 |-> \nwatch (v): p3 |-> \np1@(l_true:1) ((_ pbge 2 1 1) native-pb-0 native-pb-1)\np2@(l_true:1) + p3@(l_true:1) .w  >= 2\npropagations: 2 max_watch: 1 watch size: 2 watch-sum: 2 \n"
	if out.String() != want {
		t.Fatal("active source literal watches", out.String(), want, ls)
	}
	astTestMust(t, s.PopScopes(1))
	out.Reset()
	astTestMust(t, pb.Display(&out))
	want = "watch: p2 |-> \nwatch: p3 |-> \nwatch (v): p2 |-> p1 \nwatch (v): p3 |-> p1 \np1@(l_undef) ((_ pbge 2 1 1) native-pb-0 native-pb-1)\np2@(l_undef) + p3@(l_undef) >= 2\npropagations: 2 sum: [0:2] \n"
	if out.String() != want {
		t.Fatal("source allocated-empty lists and untrailed propagation count", out.String(), want)
	}
}

func TestSMTPBDisplayWeightedComplementAndHighWaterRecords(t *testing.T) {
	m, s, pb, atom, _ := smtPBWatchFixture(t, []int64{4, 3, 2, 2, 1}, 6, false)
	astTestMust(t, s.PushScope())
	smtPBWatchAssign(t, s, atom.Negated())
	var out bytes.Buffer
	astTestMust(t, pb.Display(&out))
	if !strings.Contains(out.String(), "(not p1)@(l_true:1) (not ((_ pbge 6 4 3 2 2 1)") || !strings.Contains(out.String(), "4*(not p2)") || !strings.Contains(out.String(), " >= 7\n") || !strings.Contains(out.String(), " .w ") {
		t.Fatal("complement source display", out.String())
	}
	astTestMust(t, s.PopScopes(1))
	astTestMust(t, s.PushScope())
	var args []ast.ASTHandle
	for _, name := range []string{"display-local-a", "display-local-b"} {
		args = append(args, smtRelevancyBool(t, m, name))
	}
	h := smtPBResolveExpression(t, m, pb.FamilyID(), args, []int32{1, 1}, 2)
	astTestMust(t, s.Internalize(h, true))
	last := int32(len(s.boolExpr) - 1)
	astTestMust(t, s.PopScopes(1))
	out.Reset()
	astTestMust(t, pb.Display(&out))
	if !strings.Contains(out.String(), fmt.Sprintf("watch (v): p%d |-> \n", last)) || strings.Contains(out.String(), "display-local") {
		t.Fatal("popped high-water watch record or stale constraint", out.String())
	}
}

type smtPBDisplayFailure struct{}

func (smtPBDisplayFailure) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSMTPBDisplayEmptyOwnerAndWriterGuards(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	s, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() {
		if !s.closed {
			astTestMust(t, s.Close())
		}
	})
	pb, err := NewSMTPseudoBooleanTheory(family)
	astTestMust(t, err)
	astTestMust(t, s.RegisterTheory(pb))
	var out bytes.Buffer
	astTestMust(t, pb.Display(&out))
	if out.Len() != 0 {
		t.Fatal("empty PB display added a heading")
	}
	if err = pb.Display(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("nil writer", err)
	}
	h := smtPBResolveExpression(t, m, pb.FamilyID(), []ast.ASTHandle{smtRelevancyBool(t, m, "error-display-a"), smtRelevancyBool(t, m, "error-display-b")}, []int32{1, 1}, 2)
	astTestMust(t, s.Internalize(h, true))
	if err = pb.Display(smtPBDisplayFailure{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("writer refusal", err)
	}
	astTestMust(t, s.Close())
	if err = pb.Display(&out); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed context display", err)
	}
}
