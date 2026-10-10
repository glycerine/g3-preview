// Source: src/util/mpz.cpp:1412-1429; src/util/mpq.cpp:142-153.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strings"
	"testing"
)

func TestRationalSMT2StreamEveryColdReservation(t *testing.T) {
	for _, tc := range [][2]string{{"-2147483648", "3"}, {"-7", "3"}, {"-123456789012345678901234567890", "1"}, {"3", "-98765432109876543210987654321"}, {"-123456789012345678901234567890", "98765432109876543210987654321"}} {
		complete := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			a := NewRational()
			serviceMust(t, m.integer.SetString(&a.Num, tc[0]))
			serviceMust(t, m.integer.SetString(&a.Den, tc[1]))
			before := a
			var out OwnedStringStream
			serviceMust(t, out.Construct(b))
			serviceMust(t, out.AppendCString("prefix:"))
			b.FailAfter(ordinal)
			body := m.rational.DisplaySMT2Into(&out, &a, true)
			s := &m.integer.smt2Stream
			integerPhysicalAudit(t, m.integer, &a.Num, &a.Den, &s.tmp)
			streamBytes := out.Buffer.String.StorageBytes()
			if a != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+streamBytes {
				t.Fatal("immediate SMT2 stream graph", tc, ordinal, body, b.Used(), streamBytes)
			}
			got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
			serviceMust(t, e)
			if !strings.HasPrefix(got, "prefix:") {
				t.Fatal("borrowed prefix discarded", got)
			}
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected stream failure", body)
				}
				if e := m.rational.DisplaySMT2Into(&out, &a, true); !errors.Is(e, ErrNumericPending) {
					t.Fatal("stopped rational replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.rational.CleanupSMT2Stream())
			serviceMust(t, m.integer.CleanupSMT2Stream())
			if after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String(); e != nil || after != got {
				t.Fatal("cleanup changed borrowed stream", after, e)
			}
			out.Buffer.OwnAbort()
			serviceMust(t, m.rational.Del(&a))
			if e := m.Close(); e != nil {
				z := m.integer
				if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
					t.Fatal("unexpected owner source stop", e)
				}
				index := int(z.SourceFinalizeStage-2) / 2
				ref := z.Tmp[index]
				if z.SourceFinalizeStage%2 == 1 {
					ref = z.Arg[index]
				}
				if ref != 0 || s.active || m.rational.smt2Stream.active {
					t.Fatal("unproved scratch hole", ref)
				}
				serviceMust(t, m.Close())
			}
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("stream refunds", b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("stream reservation horizon", tc)
		}
	}
}

func TestRationalSMT2StreamWarmAndLaterChild(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(-7), Den: NewInt(3)}
	var first OwnedStringStream
	serviceMust(t, first.Construct(b))
	serviceMust(t, m.rational.DisplaySMT2Into(&first, &a, false))
	first.Buffer.OwnAbort()
	used, state := b.Used(), r.State()
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	b.FailAfter(0)
	serviceMust(t, m.rational.DisplaySMT2Into(&out, &a, false))
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "(/ (- 7) 3)" || b.Used() != used || r.State() != state {
		t.Fatal("warm stream allocated", got, b.Used(), used)
	}
	b.FailAfter(-1)
	var later OwnedStringStream
	serviceMust(t, later.Construct(b))
	v := NewInt(-9)
	serviceMust(t, m.integer.DisplaySMT2Into(&later, &v, false))
	invocation := m.integer.smt2Stream.invocation
	serviceMust(t, m.rational.CleanupSMT2Stream())
	if m.integer.smt2Stream.invocation != invocation {
		t.Fatal("ended rational touched later integer child")
	}
	out.Buffer.OwnAbort()
	later.Buffer.OwnAbort()
}

// An ostream exception after a complete numeral insertion must preserve the
// prefix and source-destruct the negative temporary, without destroying out.
type smt2FailingStream struct {
	inner       *OwnedStringStream
	calls, fail int
	failure     error
}

func (s *smt2FailingStream) next() error {
	s.calls++
	if s.calls == s.fail {
		return s.failure
	}
	return nil
}
func (s *smt2FailingStream) AppendCString(v string) error {
	if e := s.next(); e != nil {
		return e
	}
	return s.inner.AppendCString(v)
}
func (s *smt2FailingStream) AppendInt32(v int32) error {
	if e := s.next(); e != nil {
		return e
	}
	return s.inner.AppendInt32(v)
}
func (s *smt2FailingStream) AppendNumeralText(v string) error {
	if e := s.next(); e != nil {
		return e
	}
	return s.inner.AppendNumeralText(v)
}

func TestRationalSMT2StreamEveryInsertionException(t *testing.T) {
	for fail := 1; fail <= 11; fail++ {
		m, _, b := mpfNumeralHelpersFixture(t)
		a := Rational{Num: NewInt(-7), Den: NewInt(-3)}
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		boom := errors.New("actual ostream insertion failure")
		stream := smt2FailingStream{inner: &out, fail: fail, failure: boom}
		body := m.rational.DisplaySMT2Into(&stream, &a, true)
		if !errors.Is(body, boom) {
			t.Fatal("selected insertion not reached", fail, stream.calls, body)
		}
		if m.integer.smt2Stream.tmpLive {
			t.Fatal("negative scoped numeral did not source-destruct", fail)
		}
		before, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		prefixes := []string{"", "(/ ", "(/ (- ", "(/ (- 7", "(/ (- 7.0", "(/ (- 7.0)", "(/ (- 7.0) ", "(/ (- 7.0) (- ", "(/ (- 7.0) (- 3", "(/ (- 7.0) (- 3.0", "(/ (- 7.0) (- 3.0)"}
		if before != prefixes[fail-1] {
			t.Fatal("source insertion prefix", fail, before)
		}
		serviceMust(t, m.rational.CleanupSMT2Stream())
		after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if before != after || out.Buffer.String.Closed {
			t.Fatal("cleanup destroyed borrowed prefix", fail)
		}
		out.Buffer.OwnAbort()
	}
}
