// Source: src/ast/rewriter/bit_blaster/bit_blaster_params.h:22-40.
// Source author: Leonardo de Moura (2008-10-02).
// Licensed under the MIT license; see LICENSE.txt.
package bit_blaster

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSMTBitBlasterLiteralPODDefaults(t *testing.T) {
	p := NewSMTBitBlasterParams()
	if p.ExtGates || p.Quantifiers {
		t.Fatal("literal two false defaults", p)
	}
}

func TestSMTBitBlasterAllFourScalarValuesDisplayFlagsAndOutputPrefixes(t *testing.T) {
	if reflect.TypeOf(SMTBitBlasterParams{}).Size() != 2 {
		t.Fatal("actual inline two bools, no wrapper or owner")
	}
	defaults := []string{"m_bb_ext_gates=0\nm_bb_quantifiers=0\n", "m_bb_ext_gates=1\nm_bb_quantifiers=0\n", "m_bb_ext_gates=0\nm_bb_quantifiers=1\n", "m_bb_ext_gates=1\nm_bb_quantifiers=1\n"}
	positive := []string{"m_bb_ext_gates=+0\nm_bb_quantifiers=+0\n", "m_bb_ext_gates=+1\nm_bb_quantifiers=+0\n", "m_bb_ext_gates=+0\nm_bb_quantifiers=+1\n", "m_bb_ext_gates=+1\nm_bb_quantifiers=+1\n"}
	for bits := 0; bits < 4; bits++ {
		p := SMTBitBlasterParams{bits&1 != 0, bits&2 != 0}
		for show := 0; show < 2; show++ {
			b := arena.NewBudget(^uint64(0))
			out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
			paramsMust(t, e)
			paramsMust(t, out.SetShowPos(show != 0))
			paramsMust(t, p.Display(out))
			got, e := out.String()
			paramsMust(t, e)
			want := defaults[bits]
			if show != 0 {
				want = positive[bits]
			}
			if got != want {
				t.Fatal("literal separate chains", bits, show, got, want)
			}
			paramsMust(t, out.Close())
			if b.Used() != 0 {
				t.Fatal("only actual output owns storage")
			}
		}
	}
	p := NewSMTBitBlasterParams()
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(^uint64(0))
		out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		b.FailAfter(ordinal)
		cause := p.Display(out)
		got, e := out.String()
		paramsMust(t, e)
		if len(got) > len(defaults[0]) || got != defaults[0][:len(got)] {
			t.Fatal("literal insertion refusal prefix", ordinal, got)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		stats := out.Stats()
		if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
			t.Fatal("actual output graph before fault lift", ordinal, b.Used(), stats)
		}
		b.FailAfter(-1)
		paramsMust(t, out.Close())
		if b.Used() != 0 {
			t.Fatal("output refund")
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 0 {
		t.Fatal("two short chains fit actual initial64-byte output storage", first)
	}
	b := arena.NewBudget(^uint64(0))
	file, e := os.CreateTemp(".", "bitblaster-output-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, p.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("nonthrowing file badbit")
	}
	paramsMust(t, out.Close())
	if b.Used() != 0 {
		t.Fatal("no scalar owner allocations")
	}
}
