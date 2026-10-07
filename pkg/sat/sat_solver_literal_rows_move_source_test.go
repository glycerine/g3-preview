// Source: src/util/vector.h:68-94,169-170; actual nested move pointer reads.
package sat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Guarded source pointer domains, not valid allocation exhaustion. New outer
// size is published before moves; an invalid reached old pointer must stop
// before its nulling/free and preserve the genuinely moved earlier prefix.
func TestSATConsequenceRowsExpansionReadsActualOldPointer(t *testing.T) {
	for _, tc := range []struct {
		index uint32
		word  uint64
	}{{0, 0}, {1, 0}, {1, 4}} {
		t.Run(fmt.Sprintf("%d-%d", tc.index, tc.word), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
			satOK(t, e)
			satOK(t, out.appendLiterals([]Literal{2}))
			satOK(t, out.appendLiterals([]Literal{4}))
			old := out.raw
			original := [2]util.Block{out.rows[0].Raw.Block, out.rows[1].Raw.Block}
			satOK(t, out.runtime.WriteUint32(old, 8+uint64(tc.index)*8+tc.word, 0))
			e = out.appendLiterals([]Literal{6})
			if !errors.Is(e, ErrSATOwner) {
				t.Errorf("source move ignored actual old pointer: %v", e)
			}
			if out.oldRaw != old || out.raw == old || len(out.oldRows) != 2 {
				t.Errorf("move consumed old source ownership before pointer read")
			} else {
				for i := uint32(0); i < 2; i++ {
					if i < tc.index {
						if out.rows[i].Raw.Block != original[i] || out.oldRows[i].Raw.Block != (util.Block{}) {
							t.Error("lost actual moved prefix", i)
						}
						slot, er := out.runtime.ReadUint32(old, 8+uint64(i)*8)
						satOK(t, er)
						gen, er := out.runtime.ReadUint32(old, 12+uint64(i)*8)
						satOK(t, er)
						if slot != 0 || gen != 0 {
							t.Error("completed move did not null source", i)
						}
					} else if out.oldRows[i].Raw.Block != original[i] || out.rows[i].Raw.Block != (util.Block{}) {
						t.Error("consumed unmoved source suffix", i)
					}
				}
			}
			satFullSolverRuntimeAudit(t, s, out)
			if er := out.CloseChecked(); er != nil {
				satOK(t, out.CleanupStopped())
			}
			satFullSolverRuntimeAudit(t, s, out)
		})
	}
}
