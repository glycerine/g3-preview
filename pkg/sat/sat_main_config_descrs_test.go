package sat

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Frozen descriptor bytes from sat_params.pyg and mk_genfile_common.py
// max_memory_param/pyg_default. Runtime tests do not need the C++ source tree.
const satMainConfigDescriptorFixture = `max_memory	0	4294967295	maximum amount of memory in megabytes
phase	4	caching	phase selection strategy: always_false, always_true, caching, random
phase.caching.on	0	400	phase caching on period (in number of conflicts)
phase.caching.off	0	100	phase caching off period (in number of conflicts)
restart	4	luby	restart strategy: luby or geometric
restart.initial	0	100	initial restart (number of conflicts)
restart.max	0	4294967295	maximal number of restarts.
restart.factor	2	1.5	restart increment factor for geometric strategy
random_freq	2	0.01	frequency of random case splits
random_seed	0	0	random seed
burst_search	0	100	number of conflicts before first global simplification
max_conflicts	0	4294967295	maximum number of conflicts
gc	4	glue_psm	garbage collection strategy: psm, glue, glue_psm, dyn_psm
gc.initial	0	20000	learned clauses garbage collection frequence
gc.increment	0	500	increment to the garbage collection threshold
gc.small_lbd	0	3	learned clauses with small LBD are never deleted (only used in dyn_psm)
gc.k	0	7	learned clauses that are inactive for k gc rounds are permanently deleted (only used in dyn_psm)
minimize_lemmas	1	true	minimize learned clauses
dyn_sub_res	1	true	dynamic subsumption resolution for minimizing learned clauses
core.minimize	1	false	minimize computed core
core.minimize_partial	1	false	apply partial (cheap) core minimization
parallel_threads	0	1	number of parallel threads to use
dimacs.core	1	false	extract core from DIMACS benchmarks
dimacs.display	1	false	display SAT instance in DIMACS format and return unknown instead of solving`

func TestSATMainConfigGeneratedDescriptorBytesAndOrder(t *testing.T) {
	sy := symbolManagerTest(t, nil, false)
	d := pdnew(t, sy, nil)
	paramsMust(t, collectSATConfigParamDescrs(d))
	rows := strings.Split(satMainConfigDescriptorFixture, "\n")
	n, e := d.Size()
	paramsMust(t, e)
	if n != 24 || len(rows) != 24 {
		t.Fatal(n, len(rows))
	}
	for i, row := range rows {
		f := strings.Split(row, "\t")
		k, e := d.ParamName(uint32(i))
		paramsMust(t, e)
		spelling, yes, e := sy.BareString(k)
		paramsMust(t, e)
		if !yes || spelling != f[0] {
			t.Fatal(i, spelling, f[0])
		}
		kind, e := d.KindName(f[0])
		paramsMust(t, e)
		wantKind := map[string]util.ParamKind{"0": util.ParamUInt, "1": util.ParamBool, "2": util.ParamDouble, "4": util.ParamSymbol}[f[1]]
		if kind != wantKind {
			t.Fatal(i, kind, wantKind)
		}
		value, present, e := d.DefaultName(f[0])
		paramsMust(t, e)
		if !present || value != f[2] {
			t.Fatal(i, value, f[2])
		}
		value, present, e = d.DescriptionName(f[0])
		paramsMust(t, e)
		if !present || value != f[3] {
			t.Fatal(i, value, f[3])
		}
		value, present, e = d.ModuleName(f[0])
		paramsMust(t, e)
		if !present || value != "sat" {
			t.Fatal(i, value)
		}
	}
	for _, name := range []string{"simplify_mult1", "simplify_mult2", "simplify_max"} {
		yes, e := d.ContainsName(name)
		paramsMust(t, e)
		if yes {
			t.Fatal("unexposed parameter", name)
		}
	}
}
func TestSATMainConfigGeneratedDescriptorDuplicateSourceInsert(t *testing.T) {
	sy := symbolManagerTest(t, nil, false)
	d := pdnew(t, sy, nil)
	pdinsert(t, d, "phase", util.ParamSymbol, pdstr("old"), pdstr("false"), pdstr("borrowed-original"))
	paramsMust(t, collectSATConfigParamDescrs(d))
	paramsMust(t, collectSATConfigParamDescrs(d))
	n, e := d.Size()
	paramsMust(t, e)
	if n != 24 {
		t.Fatal(n)
	}
	kind, e := d.KindName("phase")
	paramsMust(t, e)
	if kind != util.ParamSymbol {
		t.Fatal("source insert overwrote", kind)
	}
	description, yes, e := d.DescriptionName("phase")
	paramsMust(t, e)
	if !yes || description != "old" {
		t.Fatal(description, yes)
	}
	first, e := d.ParamName(0)
	paramsMust(t, e)
	spelling, _, e := sy.BareString(first)
	paramsMust(t, e)
	if spelling != "phase" {
		t.Fatal(spelling)
	}
}
func TestSATMainConfigGeneratedDescriptorFailurePrefixes(t *testing.T) {
	sy := symbolManagerTest(t, nil, false)
	failures, successes, partial := 0, 0, 0
	for fail := int64(0); fail < 180; fail++ {
		b := arena.NewBudget(^uint64(0))
		d := pdnew(t, sy, b)
		b.FailAfter(fail)
		e := collectSATConfigParamDescrs(d)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		st := d.Stats()
		if b.Used() != pdtotal(st) || st.Live > 24 || st.Live < st.Names || st.Live > st.Names+1 || st.Text.Live != 3*st.Live {
			t.Fatal("immediate source prefix graph", fail, e, st, b.Used())
		}
		if e == nil {
			successes++
		} else {
			failures++
			if st.Live > 0 {
				partial++
			}
		}
		b.FailAfter(-1)
		paramsMust(t, d.Close())
		if b.Used() != 0 {
			t.Fatal("refund", fail, b.Used())
		}
	}
	if failures == 0 || successes == 0 || partial == 0 {
		t.Fatal(failures, successes, partial)
	}
}
func TestSATMainConfigGeneratedDescriptorLifetime(t *testing.T) {
	if e := collectSATConfigParamDescrs(nil); !errors.Is(e, util.ErrParamDescrsPrecondition) {
		t.Fatal(e)
	}
	sy := symbolManagerTest(t, nil, false)
	d := pdnew(t, sy, nil)
	paramsMust(t, d.Close())
	if e := collectSATConfigParamDescrs(d); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSATMainConfigGeneratedDescriptorWarmDuplicateAllocations(t *testing.T) {
	sy := symbolManagerTest(t, nil, false)
	b := arena.NewBudget(^uint64(0))
	d := pdnew(t, sy, b)
	paramsMust(t, collectSATConfigParamDescrs(d))
	before := d.Stats()
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := collectSATConfigParamDescrs(d); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("duplicate collector host allocation", n)
	}
	if d.Stats() != before {
		t.Fatal("duplicate changed physical graph")
	}
	b.FailAfter(-1)
}
