package g3

import (
	"math"
	"sync"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

// Added source-derived tests: rlimit has no dedicated source assertion suite.
func TestResourceLimitSourceScopes(t *testing.T) {
	r, err := newUtilReslimit(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	if !r.inc() || !r.incOffset(4) || r.count() != 5 {
		t.Fatal("unlimited count")
	}
	if err := r.push(3); err != nil {
		t.Fatal(err)
	}
	if !r.incOffset(3) || r.inc() || r.count() != 9 {
		t.Fatal("inclusive limit")
	}
	r.pop()
	if r.count() != 8 || !r.inc() {
		t.Fatal("pop clamps to inner limit")
	}
	if err := r.push(10); err != nil {
		t.Fatal(err)
	}
	if err := r.push(0); err != nil {
		t.Fatal(err)
	}
	// Source min(outer limit, zero) drops the outer bound while nested.
	if !r.incOffset(100) {
		t.Fatal("source zero nested bound")
	}
	r.pop()
	if r.incOffset(0) {
		t.Fatal("outer bound restored without zero-bound clamp")
	}
	r.pop()
	if r.count() != 19 {
		t.Fatal(r.count())
	}
	r.steps = math.MaxUint64 - 1
	if err := r.push(3); err != nil {
		t.Fatal(err)
	}
	if r.limit != 0 || !r.incOffset(3) || r.count() != 1 {
		t.Fatal("source uint64 wrap")
	}
	r.pop()
}

func TestResourceLimitSourceCancellation(t *testing.T) {
	var parent, child, grand utilReslimit
	if err := parent.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	if err := child.pushChild(&grand); err != nil {
		t.Fatal(err)
	}
	parent.cancel()
	parent.incCancel()
	if !parent.cancelFlag() || !child.cancelFlag() || !grand.cancelFlag() || grand.inc() {
		t.Fatal("propagation")
	}
	parent.decCancel()
	if grand.cancelCount.Load() != 1 {
		t.Fatal("nested cancel count")
	}
	parent.resetCancel()
	parent.decCancel()
	if grand.cancelFlag() || !grand.inc() {
		t.Fatal("reset or zero decrement")
	}
	parent.cancel()
	if err := child.push(0); err != nil {
		t.Fatal(err)
	}
	if child.cancelFlag() || !grand.cancelFlag() {
		t.Fatal("push clears local cancel only")
	}
	child.pop()
	parent.cancelCount.Store(math.MaxUint32)
	parent.incCancel()
	if parent.cancelFlag() || child.cancelFlag() || grand.cancelFlag() {
		t.Fatal("source cancel unsigned wrap")
	}
	parent.popChild()
	parent.cancel()
	if child.cancelFlag() {
		t.Fatal("popped child detached")
	}
	if parent.cancelMessage() != "canceled" || child.cancelMessage() != "max. resource limit exceeded" {
		t.Fatal("source messages")
	}
	parent.close()
	child.close()
	grand.close()
}

func TestResourceLimitSourceRAII(t *testing.T) {
	var r, child utilReslimit
	scope, err := newUtilLimitScope(&r, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !r.inc() || r.inc() {
		t.Fatal("scope")
	}
	scope.close()
	scope.close()
	if r.count() != 1 || !r.inc() {
		t.Fatal("scope closes once")
	}
	r.cancel()
	outer := newUtilSuspendLimit(&r)
	inner := newUtilSuspendLimit(&r)
	if r.cancelFlag() || !r.inc() {
		t.Fatal("suspended cancel")
	}
	inner.close()
	// Source destructor always clears suspend; nesting does not restore it.
	if !r.cancelFlag() || r.inc() {
		t.Fatal("source nested suspend defect")
	}
	outer.close()
	children := newUtilChildLimits(&r)
	if err := children.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	children.close()
	children.close()
	r.resetCancel()
	r.cancel()
	if child.cancelFlag() {
		t.Fatal("child scope cleanup")
	}
	r.close()
	child.close()
}

func TestResourceLimitIndependentStackOracle(t *testing.T) {
	var r utilReslimit
	defer r.close()
	var count, bound uint64
	var stack []uint64
	state := uint32(17)
	for step := 0; step < 4000; step++ {
		state = state*1664525 + 1013904223
		switch state % 3 {
		case 0:
			if len(stack) == 20 {
				continue
			}
			delta := uint64(state>>16) % 20
			stack = append(stack, bound)
			candidate := count + delta
			if delta == 0 {
				candidate = 0
			}
			if bound == 0 || candidate < bound {
				bound = candidate
			}
			if err := r.push(uint32(delta)); err != nil {
				t.Fatal(err)
			}
		case 1:
			if len(stack) == 0 {
				continue
			}
			if bound != 0 && count > bound {
				count = bound
			}
			bound = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			r.pop()
		case 2:
			offset := uint32(state>>20) % 9
			count += uint64(offset)
			if r.incOffset(offset) != (bound == 0 || count <= bound) {
				t.Fatal(step, "admission")
			}
		}
		if r.count() != count || r.limit != bound {
			t.Fatal(step, "stack state")
		}
	}
}

func TestResourceLimitBudgetsAndWarmScopes(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	b.FailAfter(0)
	if r, err := newUtilReslimit(b); err == nil || r != nil || b.Used() != 0 {
		t.Fatal("constructor rollback")
	}
	b.FailAfter(-1)
	r, err := newUtilReslimit(b)
	if err != nil {
		t.Fatal(err)
	}
	before := b.Used()
	r.cancel()
	b.FailAfter(0)
	if err := r.push(5); err == nil || r.limit != 0 || !r.cancelFlag() || b.Used() != before {
		t.Fatal("failed push mutation")
	}
	var child utilReslimit
	if err := r.pushChild(&child); err == nil || len(r.children) != 0 || b.Used() != before {
		t.Fatal("failed child publication")
	}
	b.FailAfter(-1)
	if err := r.push(5); err != nil {
		t.Fatal(err)
	}
	r.pop()
	if err := r.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	r.popChild()
	if a := testing.AllocsPerRun(100, func() {
		if err := r.push(2); err != nil {
			panic(err)
		}
		r.inc()
		r.pop()
		if err := r.pushChild(&child); err != nil {
			panic(err)
		}
		r.cancel()
		r.resetCancel()
		r.popChild()
	}); a != 0 {
		t.Fatal(a)
	}
	r.close()
	r.close()
	if b.Used() != 0 {
		t.Fatal("close leaked", b.Used())
	}
}

func TestResourceLimitConcurrentCancel(t *testing.T) {
	var r, child utilReslimit
	if err := r.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	defer r.close()
	defer child.close()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				r.incCancel()
				child.cancelFlag()
				r.decCancel()
			}
		}()
	}
	for i := 0; i < 1000; i++ {
		r.inc()
	}
	wg.Wait()
	if r.cancelFlag() || child.cancelFlag() {
		t.Fatal("balanced concurrent cancellation")
	}
}

func TestResourceLimitSuspensionAndLateChild(t *testing.T) {
	var r, child utilReslimit
	defer r.close()
	defer child.close()
	if err := r.push(1); err != nil {
		t.Fatal(err)
	}
	if !r.inc() || r.inc() || r.cancelFlag() {
		t.Fatal("exhaustion is separate from cancel flag")
	}
	s := newUtilSuspendLimit(&r)
	if !r.incOffset(10) || r.count() != 12 {
		t.Fatal("suspension bypasses bound, retains work")
	}
	r.cancel()
	if r.cancelFlag() || r.cancelMessage() != "canceled" {
		t.Fatal("message ignores suspension")
	}
	if err := r.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	if child.cancelFlag() {
		t.Fatal("registration does not inherit previous cancellation")
	}
	r.incCancel()
	if !child.cancelFlag() || child.cancelCount.Load() != 2 {
		t.Fatal("next propagation overwrites child count")
	}
	s.close()
	r.pop()
	if r.cancelFlag() || !child.cancelFlag() || r.count() != 1 {
		t.Fatal("pop clamps and clears local flag only")
	}
	r.popChild()
}

func TestResourceLimitGrowthFailureAndPeak(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, err := newUtilReslimit(b)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := r.push(9); err != nil {
			t.Fatal(err)
		}
	}
	r.cancel()
	before := b.Used()
	b.FailAfter(0)
	if err := r.push(3); err == nil || len(r.limits) != 2 || r.limit != 9 || !r.cancelFlag() || b.Used() != before {
		t.Fatal("growth failure changed limit")
	}
	b.FailAfter(-1)
	r.close()
	// Constructor header plus an existing two-entry vector fits, replacement
	// plus the old backing store does not. This checks peak growth charging.
	header := before - 16
	b2 := arena.NewBudget(header + 24)
	r, err = newUtilReslimit(b2)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.push(1); err != nil {
		t.Fatal(err)
	}
	if err := r.push(1); err != nil {
		t.Fatal(err)
	}
	if err := r.push(1); err != arena.ErrBudget || b2.Used() != header+16 {
		t.Fatal("old+new vector peak", err, b2.Used())
	}
	r.pop()
	r.pop()
	r.close()
	if b.Used() != 0 || b2.Used() != 0 {
		t.Fatal("budget leak")
	}
}
