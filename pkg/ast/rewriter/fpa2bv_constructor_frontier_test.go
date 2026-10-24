package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFPA2BVRewriterConstructorAllocationFrontiers(t *testing.T) {
	success := false
	for stop := int64(0); stop < 128 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			f := newFPA2BVFixture(t)
			before := f.m.Budget().Used()
			f.m.Budget().FailAfter(stop)
			r, err := NewFPA2BVRewriter(f.m, f.converter)
			f.m.Budget().FailAfter(-1)
			if r != nil {
				astTestMust(t, r.Close())
			}
			if f.m.Budget().Used() != before {
				t.Fatal("partial traversal storage retained", f.m.Budget().Used(), before)
			}
			if err == nil {
				success = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal(err)
			}
		})
		if t.Failed() {
			t.Fatal("FPA conversion rewriter constructor frontier", stop)
		}
	}
	if !success {
		t.Fatal("no successful FPA rewriter constructor frontier")
	}
}
