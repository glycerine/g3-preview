package numeral

// Source: src/util/buffer.h:29-75,90-96,146-151;
// src/ast/rewriter/poly_rewriter_def.h:570-586.

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalBuffer16CopiesAndSliceGrowth(t *testing.T) {
	s, budget := serviceNew(t)
	x := serviceValue(t, s, "123456789012345678901234567890123456789/998877665544332211")
	defer func() { serviceMust(t, s.Del(&x)) }()
	b, e := NewRationalBuffer16(s)
	serviceMust(t, e)
	defer func() { serviceMust(t, b.Close()) }()
	var raw [16]Rational
	for i := uint32(0); i < 16; i++ {
		serviceMust(t, b.PushCopy(&x))
		v, e := b.At(i)
		serviceMust(t, e)
		if v.Raw.Num.Cell == x.Raw.Num.Cell || v.Raw.Den.Cell == x.Raw.Den.Cell {
			t.Fatal("append borrowed numeric ownership")
		}
		raw[i] = v.Raw
	}
	if b.count != 16 || len(b.values) != 16 || cap(b.values) < 16 {
		t.Fatal("typed slice prefix", b.count, len(b.values), cap(b.values))
	}
	serviceMust(t, s.SetInt64(&x, 7))
	used := budget.Used()
	budget.FailAfter(0)
	e = b.PushCopy(&x)
	budget.FailAfter(-1)
	if e != nil || b.count != 17 || len(b.values) != 17 || budget.Used() != used {
		t.Fatal("Go slice growth touched arena", e, b.count, budget.Used(), used)
	}
	for i := uint32(0); i < 16; i++ {
		v, e := b.At(i)
		serviceMust(t, e)
		if v.Raw != raw[i] {
			t.Fatal("slice growth changed numeral", i)
		}
		serviceText(t, s, v, "13717421001371742100137174210013717421/110986407282703579")
	}
	v, e := b.At(16)
	serviceMust(t, e)
	serviceText(t, s, v, "7")
	for i := 17; i < 32; i++ {
		serviceMust(t, b.PushCopy(&x))
	}
	serviceMust(t, b.PushCopy(&x))
	if b.count != 33 || len(b.values) != 33 {
		t.Fatal("second slice growth", b.count, len(b.values))
	}
	for i := uint32(0); i < 16; i++ {
		v, e := b.At(i)
		serviceMust(t, e)
		if v.Raw != raw[i] {
			t.Fatal("second relocation changed numeral", i)
		}
	}
	serviceMust(t, b.SourceEnd())
	serviceMust(t, b.SourceEnd())
	if b.sourceCursor != 33 || b.values != nil {
		t.Fatal("source destruction retained buffer")
	}
	if _, e = b.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ended buffer remained readable", e)
	}
}

func TestRationalBuffer16SourceDestructionStopsBeforeLaterElements(t *testing.T) {
	s, _ := serviceNew(t)
	x := serviceValue(t, s, "100000000000000000000000000000000000003")
	defer func() { serviceMust(t, s.Del(&x)) }()
	b, e := NewRationalBuffer16(s)
	serviceMust(t, e)
	defer func() { serviceMust(t, b.Close()) }()
	for i := 0; i < 3; i++ {
		serviceMust(t, b.PushCopy(&x))
	}
	// Checked stale-owner domain at the second element, not a native fatal proof.
	dead := serviceValue(t, s, "200000000000000000000000000000000000003")
	stale := dead.Raw.Num
	serviceMust(t, s.Del(&dead))
	saved := b.values[1].Raw.Num
	b.values[1].Raw.Num = stale
	third := b.values[2].Raw
	e = b.SourceEnd()
	if e == nil || b.sourceCursor != 1 || b.values[0] != (OwnedRational{}) || b.values[2].Raw != third {
		t.Fatal("source destruction did not stop ascending", e)
	}
	b.values[1].Raw.Num = saved
	if e2 := b.SourceEnd(); e2 != e {
		t.Fatal("source failure history changed", e2, e)
	}
	serviceMust(t, b.Close())
	if b.sourceCause != e {
		t.Fatal("owning cleanup replaced source history")
	}
	body, source, own := b.Causes()
	if body != nil || source != e || own != nil {
		t.Fatal("ended caller lost source destruction history", body, source, own)
	}
}

func TestRationalBuffer16ConstructionAndShallowCopyGuards(t *testing.T) {
	s, budget := serviceNew(t)
	used := budget.Used()
	budget.FailAfter(0)
	b, e := NewRationalBuffer16(s)
	budget.FailAfter(-1)
	if e != nil || b == nil || budget.Used() != used {
		t.Fatal("Go slice constructor touched arena", e, budget.Used(), used)
	}
	copy := *b
	if e = copy.Close(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal("shallow copy accepted", e)
	}
	x, e := s.NewValue()
	serviceMust(t, e)
	budget.FailAfter(0)
	for i := 0; i < 16; i++ {
		serviceMust(t, b.PushCopy(&x))
	}
	budget.FailAfter(-1)
	if b.count != 16 || len(b.values) != 16 {
		t.Fatal("default numeral slice")
	}
	serviceMust(t, b.SourceEnd())
	serviceMust(t, b.Close())
	serviceMust(t, b.Close())
	serviceMust(t, s.Del(&x))
	if budget.Used() != used {
		t.Fatal("empty numerals or wrapper leaked")
	}
}

func TestRationalBuffer16EveryActualCopyReservation(t *testing.T) {
	complete, copyFailures := false, 0
	n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(3), 4096), big.NewInt(17))
	d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 2048), big.NewInt(3))
	want := new(big.Rat).SetFrac(n, d).String()
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s, budget := serviceNew(t)
			x := serviceValue(t, s, n.String()+"/"+d.String())
			defer func() { serviceMust(t, s.Del(&x)) }()
			small := serviceValue(t, s, "7")
			defer func() { serviceMust(t, s.Del(&small)) }()
			b, e := NewRationalBuffer16(s)
			serviceMust(t, e)
			defer func() { serviceMust(t, b.Close()) }()
			for i := 0; i < 16; i++ {
				serviceMust(t, b.PushCopy(&small))
			}
			budget.FailAfter(ordinal)
			body := b.PushCopy(&x)
			budget.FailAfter(-1)
			roots := []*Int{&x.Raw.Num, &x.Raw.Den, &small.Raw.Num, &small.Raw.Den}
			end := b.count
			if b.pending {
				end++
			}
			for i := uint32(0); i < end; i++ {
				v := b.slot(i)
				roots = append(roots, &v.Raw.Num, &v.Raw.Den)
			}
			for i := uint8(0); i < b.appendCall.scope.Count; i++ {
				roots = append(roots, &b.appendCall.scope.Roots[i].Num, &b.appendCall.scope.Roots[i].Den)
			}
			integerPhysicalAudit(t, s.Manager.Z, roots...)
			if wantBytes := s.HeaderBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z); budget.Used() != wantBytes {
				t.Fatal("actual owner graph", ordinal, budget.Used(), wantBytes)
			}
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected append refusal", body)
				}
				if b.count != 16 {
					t.Fatal("failed constructor advanced position", b.count)
				}
				copyFailures++
				if !b.pending || b.appendCall.invocation == 0 || len(b.values) != 17 {
					t.Fatal("missing exact failed constructor")
				}
				if e = b.SourceEnd(); !errors.Is(e, ErrRationalLexicalPending) {
					t.Fatal("failed body simulated source return", e)
				}
				if e = b.PushCopy(&small); !errors.Is(e, ErrRationalLexicalPending) {
					t.Fatal("failed call retried", e)
				}
			} else {
				complete = true
				v, e := b.At(16)
				serviceMust(t, e)
				serviceText(t, s, v, want)
				if v.Raw.Num.Cell == x.Raw.Num.Cell || v.Raw.Den.Cell == x.Raw.Den.Cell {
					t.Fatal("copy constructor borrowed payload")
				}
				serviceMust(t, b.SourceEnd())
			}
			bodyHistory, sourceHistory := b.bodyCause, b.sourceCause
			historyBody, historySource, _ := b.Causes()
			childBody, childSource, _ := b.appendCall.Causes()
			serviceMust(t, b.Cleanup(false))
			serviceMust(t, b.Cleanup(true))
			if b.bodyCause != bodyHistory || b.sourceCause != sourceHistory {
				t.Fatal("cleanup changed history")
			}
			if b.appendCall.invocation != 0 {
				afterBody, afterSource, _ := b.appendCall.Causes()
				if afterBody != childBody || afterSource != childSource {
					t.Fatal("child cleanup changed history")
				}
			}
			serviceMust(t, b.Close())
			endedBody, endedSource, endedOwn := b.Causes()
			if endedBody != historyBody || endedSource != historySource || endedOwn != nil {
				t.Fatal("ended owner lost append source history", endedBody, endedSource, endedOwn)
			}
		})
	}
	if !complete || copyFailures == 0 {
		t.Fatal("reservation horizon", complete, copyFailures)
	}
}
