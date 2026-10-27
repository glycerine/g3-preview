package ast

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type fpaFactorySetterObserver struct {
	bytes.Buffer
	manager   *ASTManager
	first     ASTRef
	published bool
	observed  bool
	exponent  int64
	scopes    int
}

func (w *fpaFactorySetterObserver) WriteString(body string) (int, error) {
	return w.Write([]byte(body))
}

func (w *fpaFactorySetterObserver) Write(body []byte) (int, error) {
	n, err := w.Buffer.Write(body)
	// Read the real scoped MPF at the existing numerical setter's source TRACE.
	// At entry to set(1), the reused q must still contain the preceding zero.
	if strings.HasSuffix(w.Buffer.String(), "set: value = 1\n") {
		w.observed = true
		ref, err := w.manager.reference(w.first)
		if err == nil && ref.node.slot != 0 {
			_, kind, e := w.manager.declIdentity(w.manager.appAt(ref.node.slot).decl)
			w.published = e == nil && kind == astFPAPlusZero && ref.owned
		}
		for f := w.manager.floatingPointPlugin.numeralProbeFrames; f != nil; f = f.next {
			w.scopes++
		}
		if f := w.manager.floatingPointPlugin.numeralProbeFrames; f != nil {
			w.exponent = f.value.Exponent()
		}
	}
	return n, err
}

func TestASTFPAModelFactoryUsesOneLiveNumericalScope(t *testing.T) {
	m := fpaModelTestManager(t)
	observer := &fpaFactorySetterObserver{manager: m}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	tags.Enable("mpf_dbg")
	astTestMust(t, m.floatingPointPlugin.numeric.BindTrace(tags))
	sort, err := m.FloatingPointSort(8, 24)
	astTestMust(t, err)
	firstRef, err := m.NewRef(ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&firstRef)) }()
	secondRef, err := m.NewRef(ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&secondRef)) }()
	observer.first = firstRef
	astTestMust(t, m.MkFPAModelFactoryPair(sort, firstRef, secondRef))
	first, err := m.RefGet(firstRef)
	astTestMust(t, err)
	second, err := m.RefGet(secondRef)
	astTestMust(t, err)
	if first.IsNull() || second.IsNull() || first == second {
		t.Fatal("source pair missing")
	}
	if tags.ShouldTrace("mpf_dbg") {
		if !observer.observed || !observer.published || observer.scopes != 1 || observer.exponent != -127 {
			t.Fatal("set(1) did not reuse live q containing zero", observer.observed, observer.scopes, observer.exponent, observer.String())
		}
		if strings.Count(observer.String(), "set: value = 0\n") != 1 || strings.Count(observer.String(), "set: value = 1\n") != 1 {
			t.Fatal("source integer setters missing", observer.String())
		}
	} else if observer.Len() != 0 {
		t.Fatal("erased setter trace wrote output")
	}
	if m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("factory retained its scoped MPF")
	}
}
