package numeral

import (
	"strconv"
	"strings"
	"testing"
)

type rationalRawDisplaySink struct {
	text strings.Builder
	hook func(string)
}

func (s *rationalRawDisplaySink) AppendCString(v string) error {
	if s.hook != nil {
		s.hook(v)
	}
	_, e := s.text.WriteString(v)
	return e
}
func (s *rationalRawDisplaySink) AppendNumeralText(v string) error { return s.AppendCString(v) }
func (s *rationalRawDisplaySink) AppendInt32(v int32) error {
	return s.AppendCString(strconv.FormatInt(int64(v), 10))
}

func TestRationalRawDisplayLiteralAndLateDenominatorRead(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	q := NewRational()
	for _, value := range []string{"0", "19", "-2/3", "18446744073709551619/7"} {
		serviceMust(t, m.SetString(&q, value))
		out := &rationalRawDisplaySink{}
		var receipt NumeralInsertionReceipt
		serviceMust(t, m.DisplayIntoTracked(out, &q, &receipt))
		if out.text.String() != value || receipt.Active() {
			t.Fatal("raw manager display", out.text.String(), value)
		}
		serviceMust(t, m.CleanupSourceDisplay(&receipt))
	}
	serviceMust(t, m.SetString(&q, "7/11"))
	out := &rationalRawDisplaySink{hook: func(v string) {
		if v == "/" {
			q.Den = NewInt(1)
		}
	}}
	var receipt NumeralInsertionReceipt
	serviceMust(t, m.DisplayIntoTracked(out, &q, &receipt))
	if out.text.String() != "7/1" {
		t.Fatal("display reread the integer predicate or read denominator before slash callback", out.text.String())
	}
	serviceMust(t, m.Del(&q))
}
