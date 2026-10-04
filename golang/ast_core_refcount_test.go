// Licensed under the MIT license.
package g3

import (
	"math"
	"testing"
)

func TestASTSourceRefcountUint32ReleaseDebugDistinction(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(9000, b)
	astTestMust(t, e)
	ref, e := m.NewRef(h)
	astTestMust(t, e)
	r, e := m.internal(h)
	astTestMust(t, e)
	old := m.nodeAt(r.slot).rc
	m.nodeAt(r.slot).rc = math.MaxUint32
	m.budget.FailAfter(0)
	e = m.IncRef(h)
	if astSourceDebug {
		if e != ErrASTDomain || m.nodeAt(r.slot).rc != math.MaxUint32 {
			t.Fatal("source debug counter precondition", e)
		}
	} else {
		if e != nil || m.nodeAt(r.slot).rc != 0 {
			t.Fatal("source unsigned increment wrap", e)
		}
		if _, e = m.GetKind(h); e != nil {
			t.Fatal("zero wrap must not itself delete node", e)
		}
		if e = m.DecRef(h); e != ErrASTDomain {
			t.Fatal("source decrement assertion guard", e)
		}
	}
	m.nodeAt(r.slot).rc = old
	m.budget.FailAfter(-1)
	astTestMust(t, m.DelRef(&ref))
}
