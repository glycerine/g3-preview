// Source: src/util/hashtable.h:328-352,576-589; actual nested begin pointer.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package sat

import (
	"errors"
	"testing"
)

// A guarded nested pointer domain, not valid allocation failure. The source
// iterator must read the actual parent pointer at capture, before union can
// borrow table contents. Cold provenance remains an owning-abort root only.
func TestSATConsequenceIndexIteratorCaptureReadsActualNestedPointer(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.indexSet(s))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	satOK(t, f.index.insert(17))
	satOK(t, s.antecedents.insert(7, &f.index))
	row, e := s.antecedents.find(7)
	satOK(t, e)
	old := row.raw.Slot()
	satOK(t, s.runtime.memory.WriteUint32(row.parent, row.base, 0))
	_, e = row.capture()
	if !errors.Is(e, ErrSATOwner) {
		t.Error("source iterator ignored actual nested pointer", e)
	}
	e = f.index.union(row)
	if !errors.Is(e, ErrSATOwner) {
		t.Error("source union borrowed cold table after pointer alteration", e)
	}
	satFullSolverRuntimeAudit(t, s)
	// Restoring the fixture is solely for completed source destructor cleanup;
	// production capture/union must never repair the actual source pointer.
	satOK(t, s.runtime.memory.WriteUint32(row.parent, row.base, old))
}
