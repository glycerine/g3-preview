package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerAllocateSourceCapacityAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := integerManager(t)
	live := m.Cells.Stats().Live
	if ref, err := m.Allocate(3, 0); ref != 0 || !isMPNVerificationThrow(err) || m.Cells.Stats().Live != live {
		t.Fatal("Throw must precede physical allocation", ref, err)
	}
	ref, err := m.Allocate(3, 0)
	if err != nil || ref == 0 || m.Cell(ref).Capacity != 3 || !m.Cell(ref).Unpublished {
		t.Fatal("Continue must reach exact requested capacity allocation", ref, err)
	}
	serviceMust(t, d.EnableAssertions(false))
	ref2, err := m.Allocate(3, 0)
	if err != nil || ref2 == 0 || m.Cell(ref2).Capacity != 3 {
		t.Fatal("disabled source assertion must reach allocation", ref2, err)
	}
	contents, err := os.ReadFile(log.Name())
	serviceMust(t, err)
	if n := strings.Count(string(contents), "File: src/util/mpz.h\nLine: 125\ncapacity >= m_init_cell_capacity\n"); n != 2 {
		t.Fatal("source allocation assertion location and count", n, string(contents))
	}
	serviceMust(t, d.Close())
	if ref3, err := m.Allocate(6, 0); ref3 != 0 || !errors.Is(err, arena.ErrClosed) || m.Cells.Stats().Live != live+2 {
		t.Fatal("true allocation assertion must read bound owner before allocation", ref3, err)
	}
}
