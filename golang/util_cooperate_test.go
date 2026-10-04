// Licensed under the MIT license.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func cooperationTestOwner(t *testing.T) (*utilCooperation, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	c, err := newUtilCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		if b.Used() != 0 {
			t.Fatal("retained cold owner", b.Used())
		}
	})
	return c, b
}

func TestCooperationNoOMPSourceCheckpointAndSection(t *testing.T) {
	c, _ := cooperationTestOwner(t)
	if c.enabled || c.ownerThread != -1 {
		t.Fatal("source globals")
	}
	if err := c.Cooperate("unused task"); err != nil || c.ownerThread != -1 {
		t.Fatal(err, c.ownerThread)
	}
	if err := c.BeginSection(); err != nil {
		t.Fatal(err)
	}
	for _, task := range []string{"first", "second", "", "embedded\x00suffix"} {
		if err := c.Cooperate(task); err != nil || c.ownerThread != 0 {
			t.Fatal(err, c.ownerThread)
		}
	}
	if err := c.EndSection(); err != nil {
		t.Fatal(err)
	}
	if c.enabled || c.ownerThread != 0 {
		t.Fatal("section destructor must retain lock owner")
	}
	if err := c.Cooperate("disabled"); err != nil || c.ownerThread != 0 {
		t.Fatal(err, c.ownerThread)
	}
	if err := c.EndTask(); err != nil || c.ownerThread != -1 {
		t.Fatal(err, c.ownerThread)
	}
}

func TestCooperationNoOMPSourceAssertionSites(t *testing.T) {
	c, _ := cooperationTestOwner(t)
	err := c.Checkpoint("direct disabled")
	if cooperationSourceDebug {
		if !errors.Is(err, ErrCooperationEnabled) || c.ownerThread != -1 {
			t.Fatal(err, c.ownerThread)
		}
	} else if err != nil || c.ownerThread != 0 {
		t.Fatal(err, c.ownerThread)
	}
	if err = c.EndTask(); err != nil {
		t.Fatal(err)
	}
	err = c.EndSection()
	if cooperationSourceDebug && !errors.Is(err, ErrCooperationEnabled) || !cooperationSourceDebug && err != nil {
		t.Fatal(err)
	}
	if err = c.BeginSection(); err != nil {
		t.Fatal(err)
	}
	err = c.BeginSection()
	if cooperationSourceDebug && !errors.Is(err, ErrCooperationEnabled) || !cooperationSourceDebug && err != nil {
		t.Fatal(err)
	}
	if !c.enabled {
		t.Fatal("nested assertion changed flag")
	}
	err = c.BeginTask("inside section")
	if cooperationSourceDebug {
		if !errors.Is(err, ErrCooperationParallel) || c.ownerThread != -1 {
			t.Fatal(err, c.ownerThread)
		}
	} else if err != nil || c.ownerThread != 0 {
		t.Fatal(err, c.ownerThread)
	}
	if err = c.EndSection(); err != nil {
		t.Fatal(err)
	}
	err = c.BeginTask("outside section")
	if cooperationSourceDebug {
		if !errors.Is(err, ErrCooperationEnabled) || c.ownerThread != -1 {
			t.Fatal(err, c.ownerThread)
		}
	} else if err != nil || c.ownerThread != 0 {
		t.Fatal(err, c.ownerThread)
	}
}

func TestCooperationNoOMPTaskReleaseAndRuntimeEnable(t *testing.T) {
	c, _ := cooperationTestOwner(t)
	// Source task destructor releases the current thread's owner even if its
	// section has ended. A second destructor with no owner performs no action.
	for i := 0; i < 30; i++ {
		if err := c.BeginSection(); err != nil {
			t.Fatal(err)
		}
		if err := c.Checkpoint("upolynomial"); err != nil {
			t.Fatal(err)
		}
		if err := c.EndTask(); err != nil || c.ownerThread != -1 {
			t.Fatal(err, c.ownerThread)
		}
		if err := c.EndTask(); err != nil {
			t.Fatal(err)
		}
		if err := c.Checkpoint("next"); err != nil {
			t.Fatal(err)
		}
		if err := c.EndSection(); err != nil || c.ownerThread != 0 {
			t.Fatal(err, c.ownerThread)
		}
		if err := c.EndTask(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCooperationColdOwnerFaultAndCopies(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	b.FailAfter(0)
	c, err := newUtilCooperation(b)
	if c != nil || !errors.Is(err, arena.ErrInjectedFailure) || b.Used() != 0 {
		t.Fatal(c, err, b.Used())
	}
	b.FailAfter(-1)
	c, err = newUtilCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	if b.Used() != uint64(reflect.TypeOf(utilCooperation{}).Size()) {
		t.Fatal("physical header", b.Used())
	}
	copyOwner := *c
	if err = copyOwner.Close(); !errors.Is(err, ErrCooperationOwner) {
		t.Fatal(err)
	}
	var zero utilCooperation
	if err = zero.Cooperate("x"); !errors.Is(err, ErrCooperationOwner) {
		t.Fatal(err)
	}
	var absent *utilCooperation
	if err = absent.Checkpoint("x"); !errors.Is(err, ErrCooperationOwner) {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil || b.Used() != 0 {
		t.Fatal(err, b.Used())
	}
	if err = c.BeginSection(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCooperationNoOMPZeroWarmAllocations(t *testing.T) {
	c, b := cooperationTestOwner(t)
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if err := c.BeginSection(); err != nil {
			panic(err)
		}
		if err := c.Cooperate("upolynomial"); err != nil {
			panic(err)
		}
		if err := c.EndTask(); err != nil {
			panic(err)
		}
		if err := c.EndSection(); err != nil {
			panic(err)
		}
	}); n != 0 {
		t.Fatal(n)
	}
}

func TestCooperationActualCPPNoOMPReleaseTrace(t *testing.T) {
	if cooperationSourceDebug {
		t.Skip("actual C++ fixture compiled without Z3DEBUG; debug assertion sites tested separately")
	}
	c, _ := cooperationTestOwner(t)
	// Generated by compiling the actual cooperate.cpp with -D_NO_OMP_ under
	// .cache/cooperate-source/probe.cpp. Its m_task stayed nullptr at all sites.
	// Each row records action, enabled and owner after the actual source action.
	for _, row := range []struct {
		action  uint8
		enabled bool
		owner   int32
	}{
		{0, false, -1}, {1, false, -1}, {2, false, 0}, {3, false, 0},
		{4, false, -1}, {5, true, -1}, {1, true, 0}, {1, true, 0},
		{3, true, 0}, {4, true, -1}, {1, true, 0}, {6, false, 0}, {1, false, 0},
	} {
		var err error
		switch row.action {
		case 1:
			err = c.Cooperate("fixture")
		case 2:
			err = c.Checkpoint("fixture")
		case 3:
			err = c.BeginTask("fixture")
		case 4:
			err = c.EndTask()
		case 5:
			err = c.BeginSection()
		case 6:
			err = c.EndSection()
		}
		if err != nil || c.enabled != row.enabled || c.ownerThread != row.owner {
			t.Fatal(row, err, c.enabled, c.ownerThread)
		}
	}
}
