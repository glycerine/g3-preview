package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Only actual lexical frame headers are charged. Growing the native pointer
// directory preserves cached frame addresses and their owning numeral tokens.
func TestPolynomialEvaluationNativeRationalDirectory(t *testing.T) {
	m := polynomialPSCFullFixtureWithLimit(t, 1<<25)
	m.evaluation.rational = m.core.config.Rationals
	budget := m.originalBudget
	const count = 4097
	frames := make([]*polynomialEvalRationalFrame, count)
	header := uint64(reflect.TypeOf(polynomialEvalRationalFrame{}).Size())
	used := budget.Used()
	budget.FailAfter(0)
	_, e := m.evaluationRationalFrame()
	if !numericPhysicalReservationRefusal(e) || len(m.evaluation.rationalFrames) != 0 || budget.Used() != used {
		t.Fatal("cold actual frame header refusal", e)
	}
	budget.FailAfter(-1)
	var token uint64
	for i := range frames {
		used := budget.Used()
		budget.FailAfter(1)
		var e error
		frames[i], e = m.evaluationRationalFrame()
		serviceMust(t, e)
		if budget.Used()-used != header {
			t.Fatal("frame directory charged simulated storage", i, budget.Used()-used, header)
		}
		budget.FailAfter(-1)
		if i == 0 {
			serviceMust(t, m.core.integer.SetUint64(frames[0].aux.BorrowNumerator(), 1<<40|17))
			token = (frames[0].aux.BorrowNumerator()).NativeCellToken()
		}
	}
	if token == 0 || (frames[0].aux.BorrowNumerator()).NativeCellToken() != token || m.evaluation.depth != count {
		t.Fatal("frame growth lost actual BIG owner")
	}
	for i := count - 1; i >= 0; i-- {
		if frames[i] != m.evaluation.rationalFrames[i] {
			t.Fatal("cached frame address changed", i)
		}
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationRationalFrame(frames[i]))
	}
	if (frames[0].aux.BorrowNumerator()).NativeCellToken() != 0 {
		t.Fatal("frame destruction retained BIG token")
	}
	used = budget.Used()
	budget.FailAfter(0)
	for i := range frames {
		f, e := m.evaluationRationalFrame()
		serviceMust(t, e)
		if f != frames[i] || !f.constructed || f.stopped || f.stage != 0 {
			t.Fatal("cached frame default construction changed identity", i)
		}
	}
	if budget.Used() != used {
		t.Fatal("cached frame reuse reserved storage")
	}
	budget.FailAfter(-1)
	for i := count - 1; i >= 0; i-- {
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationRationalFrame(frames[i]))
	}
}

// Only actual lexical frame headers are charged. Growing the native pointer
// directory preserves cached frame addresses and their owning numeral tokens.
func TestPolynomialEvaluationNativeIntegerDirectory(t *testing.T) {
	m := polynomialPSCFullFixtureWithLimit(t, 1<<25)
	budget := m.originalBudget
	const count = 4097
	frames := make([]*polynomialEvalIntegerFrame, count)
	header := uint64(reflect.TypeOf(polynomialEvalIntegerFrame{}).Size())
	used := budget.Used()
	budget.FailAfter(0)
	_, e := m.evaluationIntegerFrame()
	if !numericPhysicalReservationRefusal(e) || len(m.evaluation.integerFrames) != 0 || budget.Used() != used {
		t.Fatal("cold actual frame header refusal", e)
	}
	budget.FailAfter(-1)
	var token uint64
	for i := range frames {
		used := budget.Used()
		budget.FailAfter(1)
		var e error
		frames[i], e = m.evaluationIntegerFrame()
		serviceMust(t, e)
		if budget.Used()-used != header {
			t.Fatal("frame directory charged simulated storage", i, budget.Used()-used, header)
		}
		budget.FailAfter(-1)
		if i == 0 {
			serviceMust(t, m.core.integer.SetUint64(&frames[0].aux, 1<<40|17))
			token = (&frames[0].aux).NativeCellToken()
		}
	}
	if token == 0 || (&frames[0].aux).NativeCellToken() != token || m.evaluation.depth != count {
		t.Fatal("frame growth lost actual BIG owner")
	}
	for i := count - 1; i >= 0; i-- {
		if frames[i] != m.evaluation.integerFrames[i] {
			t.Fatal("cached frame address changed", i)
		}
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationIntegerFrame(frames[i]))
	}
	if (&frames[0].aux).NativeCellToken() != 0 {
		t.Fatal("frame destruction retained BIG token")
	}
	used = budget.Used()
	budget.FailAfter(0)
	for i := range frames {
		f, e := m.evaluationIntegerFrame()
		serviceMust(t, e)
		if f != frames[i] || !f.constructed || f.stopped || f.stage != 0 {
			t.Fatal("cached frame default construction changed identity", i)
		}
	}
	if budget.Used() != used {
		t.Fatal("cached frame reuse reserved storage")
	}
	budget.FailAfter(-1)
	for i := count - 1; i >= 0; i-- {
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationIntegerFrame(frames[i]))
	}
}

// Only actual lexical frame headers are charged. Growing the native pointer
// directory preserves cached frame addresses and their owning numeral tokens.
func TestPolynomialEvaluationNativeIntervalDirectory(t *testing.T) {
	b, _ := binaryManager(t)
	v, e := util.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	t.Cleanup(v.Close)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{
		Integer: b.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil },
	}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	m.evaluation.interval = v
	budget := m.originalBudget
	const count = 4097
	frames := make([]*polynomialEvalIntervalFrame, count)
	header := uint64(reflect.TypeOf(polynomialEvalIntervalFrame{}).Size())
	used := budget.Used()
	budget.FailAfter(0)
	_, e = m.evaluationIntervalFrame()
	if !numericPhysicalReservationRefusal(e) || len(m.evaluation.intervalFrames) != 0 || budget.Used() != used {
		t.Fatal("cold actual frame header refusal", e)
	}
	budget.FailAfter(-1)
	var token uint64
	for i := range frames {
		used := budget.Used()
		budget.FailAfter(1)
		var e error
		frames[i], e = m.evaluationIntervalFrame()
		serviceMust(t, e)
		if budget.Used()-used != header {
			t.Fatal("frame directory charged simulated storage", i, budget.Used()-used, header)
		}
		budget.FailAfter(-1)
		if i == 0 {
			serviceMust(t, m.core.integer.SetUint64(frames[0].aux.BorrowLower().BorrowNumerator(), 1<<40|17))
			token = (frames[0].aux.BorrowLower().BorrowNumerator()).NativeCellToken()
		}
	}
	if token == 0 || (frames[0].aux.BorrowLower().BorrowNumerator()).NativeCellToken() != token || m.evaluation.depth != count {
		t.Fatal("frame growth lost actual BIG owner")
	}
	for i := count - 1; i >= 0; i-- {
		if frames[i] != m.evaluation.intervalFrames[i] {
			t.Fatal("cached frame address changed", i)
		}
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationIntervalFrame(frames[i]))
	}
	if (frames[0].aux.BorrowLower().BorrowNumerator()).NativeCellToken() != 0 {
		t.Fatal("frame destruction retained BIG token")
	}
	used = budget.Used()
	budget.FailAfter(0)
	for i := range frames {
		f, e := m.evaluationIntervalFrame()
		serviceMust(t, e)
		if f != frames[i] || !f.constructed || f.stopped || f.stage != 0 {
			t.Fatal("cached frame default construction changed identity", i)
		}
	}
	if budget.Used() != used {
		t.Fatal("cached frame reuse reserved storage")
	}
	budget.FailAfter(-1)
	for i := count - 1; i >= 0; i-- {
		m.evaluation.depth--
		serviceMust(t, m.finishEvaluationIntervalFrame(frames[i]))
	}
}
