package sat

import "testing"

func TestSATSimplifierBinaryWatchNativeStableSort(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	w, e := l.propagation.variables.watches.Get(0)
	satOK(t, e)
	binary := func(l Literal, learned bool) satWatched {
		v, e := newSATBinaryWatch(l, learned)
		satOK(t, e)
		return v
	}
	ternary, e := newSATTernaryWatch(2, 4)
	satOK(t, e)
	external, e := newSATExternalWatch(3)
	satOK(t, e)
	input := []satWatched{
		binary(4, true), ternary, binary(2, true), binary(4, false), external, binary(2, false),
	}
	for _, v := range input {
		satOK(t, w.Push(v))
	}
	before := l.budget.Used()
	l.budget.FailAfter(0)
	satOK(t, s.sortBinaryWatches(w))
	l.budget.FailAfter(-1)
	if l.budget.Used() != before {
		t.Fatal("native watch sort charged arena workspace", before, l.budget.Used())
	}
	want := []satWatched{input[5], input[2], input[3], input[0], ternary, external}
	for i, expected := range want {
		got, e := w.Read(uint32(i))
		satOK(t, e)
		if !got.Equal(expected) {
			t.Fatal("source stable binary ordering", i, got, expected)
		}
	}
}
