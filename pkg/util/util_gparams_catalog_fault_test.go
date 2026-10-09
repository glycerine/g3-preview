package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func catalogGraph(c *GlobalParamsCatalog) uint64 {
	n := c.OwnerBytes
	if c.Region != nil {
		s := c.Region.Stats()
		d := s.Descriptors
		n += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	if c.Pending != nil {
		s := c.Pending.Stats()
		n += s.OwnerBytes + s.TableBytes + s.NamesBytes + gparamsVectorGraph(s.Text)
	}
	return n
}
func TestGlobalParamsCatalogConstructorFailureAndDomains(t *testing.T) {
	_, sy, _ := paramsTestOwner(t)
	failures, successes := 0, 0
	for prefix := int64(0); prefix < 180; prefix++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(prefix)
		c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		b.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || c != nil {
				t.Fatal(prefix, c, e)
			}
			failures++
		} else {
			successes++
			if b.Used() != catalogGraph(c) {
				t.Fatal(prefix, b.Used(), catalogGraph(c))
			}
			paramsMust(t, c.Close())
		}
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	b := arena.NewBudget(^uint64(0))
	cfg := catalogTestConfig(b)
	cfg.Text.Descriptors.Budget = arena.NewBudget(100)
	if c, e := NewGlobalParamsCatalog(sy, cfg); c != nil || !errors.Is(e, arena.ErrConfig) || b.Used() != 0 {
		t.Fatal(c, e, b.Used())
	}
	c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	defer func() {
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	other, foreign, _ := paramsTestOwner(t)
	base := b.Used()
	if g, e := NewGlobalParams(foreign, other, GlobalParamsConfig{Budget: b, Catalog: c}); g != nil || !errors.Is(e, ErrGlobalParamsOwner) || b.Used() != base {
		t.Fatal(g, e, b.Used(), base)
	}
	if e = c.Register(GlobalParamsRegistrar{}); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal(e)
	}
}
func TestGlobalParamsCatalogDispatcherFaultRootsAndLateDescription(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	// Actual complete dispatcher reservation threshold, discovered by bounded
	// doubling then binary search. No mock callback or budget-counter bypass.
	run := func(prefix int64, inspect bool) (bool, bool, bool) {
		b := arena.NewBudget(^uint64(0))
		c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		g, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
		paramsMust(t, e)
		b.FailAfter(prefix)
		e = g.ensureRegistered()
		b.FailAfter(-1)
		ok := e == nil
		pending := c.Pending != nil
		late := !ok && g.ModuleDescriptors.size() == 19 && c.Pending == nil
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(prefix, e)
		}
		if !g.modulesRegistered || b.Used() != gparamsGraph(g)+catalogGraph(c) {
			t.Fatal(prefix, g.modulesRegistered, b.Used(), gparamsGraph(g), catalogGraph(c))
		}
		if inspect && pending {
			retained := c.Pending
			before := b.Used()
			// Normal check_registered never retries; root survives untouched.
			paramsMust(t, g.ensureRegistered())
			if c.Pending != retained || b.Used() != before {
				t.Fatal("registration retry destroyed prefix")
			}
			g.registering = true
			e = c.Register(GlobalParamsRegistrar{g})
			g.registering = false
			if !errors.Is(e, ErrGlobalParamsCatalogPending) || c.Pending != retained || b.Used() != before {
				t.Fatal(e, c.Pending, b.Used(), before)
			}
			paramsMust(t, c.CleanupPending())
			if c.Pending != nil || b.Used() != gparamsGraph(g)+catalogGraph(c) {
				t.Fatal("pending cleanup", b.Used())
			}
		}
		paramsMust(t, g.Close())
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
		return ok, pending, late
	}
	lo, hi := int64(-1), int64(1)
	for {
		ok, _, _ := run(hi, false)
		if ok {
			break
		}
		lo = hi
		hi *= 2
		if hi > 65536 {
			t.Fatal("finite catalog did not complete")
		}
	}
	for hi-lo > 1 {
		mid := (hi + lo) / 2
		ok, _, _ := run(mid, false)
		if ok {
			hi = mid
		} else {
			lo = mid
		}
	}
	total := hi
	pending, late := false, false
	for _, prefix := range []int64{0, 1, 2, 3, 4, total / 8, total / 4, total / 2, 3 * total / 4, total - 2, total - 1, total} {
		ok, p, l := run(prefix, true)
		pending = pending || p
		late = late || l
		if (prefix == total) != ok {
			t.Fatal("threshold", prefix, total, ok)
		}
	}
	if !pending || !late {
		t.Fatal("missing actual module and late-description prefix", total, pending, late)
	}
	t.Log("actual complete registration reserve threshold", total)
}
