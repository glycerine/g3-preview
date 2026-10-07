// Source: src/util/vector.h:68-94,169-170; actual nested move pointer reads.
package util

import (
	"errors"
	"fmt"
	"testing"
)

// Guarded source pointer domains, not valid allocation exhaustion. New outer
// size is published before moves; an invalid reached old pointer must stop
// before its nulling/free and preserve the genuinely moved earlier prefix.
func TestRuntimeUnsignedRowsExpansionReadsActualOldPointer(t *testing.T) {
	for _, tc := range []struct {
		index uint32
		word  uint64
	}{{0, 0}, {1, 0}, {1, 4}} {
		t.Run(fmt.Sprintf("%d-%d", tc.index, tc.word), func(t *testing.T) {
			m, b := memoryRuntimeFixture(t, false, 64)
			out, e := NewRuntimeUnsignedRows(m, b)
			memoryRuntimeRequire(t, e)
			memoryRuntimeRequire(t, cliqueTestAppendValues(out, []uint32{2}))
			memoryRuntimeRequire(t, cliqueTestAppendValues(out, []uint32{4}))
			old := out.StorageState().Raw
			original := [2]Block{out.StorageState().Rows[0].Block, out.StorageState().Rows[1].Block}
			memoryRuntimeRequire(t, m.WriteUint32(old, 8+uint64(tc.index)*8+tc.word, 0))
			e = cliqueTestAppendValues(out, []uint32{6})
			if !errors.Is(e, ErrRuntimeVectorOwner) {
				t.Errorf("source move ignored actual old pointer: %v", e)
			}
			if out.StorageState().OldRaw != old || out.StorageState().Raw == old || len(out.StorageState().OldRows) != 2 {
				t.Errorf("move consumed old source ownership before pointer read")
			} else {
				for i := uint32(0); i < 2; i++ {
					if i < tc.index {
						if out.StorageState().Rows[i].Block != original[i] || out.StorageState().OldRows[i].Block != (Block{}) {
							t.Error("lost actual moved prefix", i)
						}
						slot, er := m.ReadUint32(old, 8+uint64(i)*8)
						memoryRuntimeRequire(t, er)
						gen, er := m.ReadUint32(old, 12+uint64(i)*8)
						memoryRuntimeRequire(t, er)
						if slot != 0 || gen != 0 {
							t.Error("completed move did not null source", i)
						}
					} else if out.StorageState().OldRows[i].Block != original[i] || out.StorageState().Rows[i].Block != (Block{}) {
						t.Error("consumed unmoved source suffix", i)
					}
				}
			}
			cliqueTestRowsAudit(t, m, out)
			if er := out.CloseChecked(); er != nil {
				memoryRuntimeRequire(t, out.CleanupStopped())
			}
			cliqueTestRowsAudit(t, m, out)
		})
	}
}
