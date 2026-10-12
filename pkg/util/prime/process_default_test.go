package prime

import "testing"

func TestProcessDefaultPrimeIteratorEagerSharedCache(t *testing.T) {
	first, e := BorrowProcessGenerator()
	if e != nil || first == nil || first.Stats().Count <= 2 {
		t.Fatal("eager default generator", first, e)
	}
	second, e := BorrowProcessGenerator()
	if e != nil || second != first {
		t.Fatal("process generator identity", e)
	}
	a, e := NewServiceIterator(nil, nil)
	if e != nil || !a.global || a.generator != first {
		t.Fatal("source null iterator binding", e)
	}
	b, e := NewServiceIterator(nil, nil)
	if e != nil || b.generator != first {
		t.Fatal("second source null iterator binding", e)
	}
	for _, step := range []struct {
		it   *ServiceIterator
		want uint64
	}{{&a, 2}, {&a, 3}, {&b, 2}, {&a, 5}, {&b, 3}} {
		got, e := step.it.Next()
		if e != nil || got != step.want {
			t.Fatalf("independent iterator index over shared cache: got %d, want %d, error %v", got, step.want, e)
		}
	}
}
