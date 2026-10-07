package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func pdStreamBuffer(t *testing.T, b *arena.Budget) *StringBuffer64 {
	t.Helper()
	s, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, s.Close()) })
	return s
}
func pdStreamText(t *testing.T, b *StringBuffer64) string {
	t.Helper()
	s, e := b.CStr()
	paramsMust(t, e)
	return s
}
func TestParamDescrsDisplayIntoSourceBranchesAndSort(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	pdinsert(t, d, "Z_key", ParamUInt, pdstr("Z text\x00ignored"), pdstr(""), nil)
	pdinsert(t, d, "a-key", ParamBool, pdstr("A text"), pdstr("false"), nil)
	pdinsert(t, d, "middle", ParamSymbol, pdstr("M text"), nil, nil)
	for _, smt2 := range []bool{false, true} {
		for _, descr := range []bool{false, true} {
			b := pdStreamBuffer(t, nil)
			paramsMust(t, b.AppendCString("prefix\n"))
			want, e := d.Display(2, smt2, descr)
			paramsMust(t, e)
			paramsMust(t, d.DisplayInto(b, 2, smt2, descr))
			if got := pdStreamText(t, b); got != "prefix\n"+want {
				t.Fatal(smt2, descr, got, want)
			}
		}
	}
}
func TestParamDescrsDisplayIntoNullDescriptionActualAction(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	pdinsert(t, d, "A-key", ParamUInt, nil, nil, nil)
	for _, descr := range []bool{false, true} {
		b := pdStreamBuffer(t, nil)
		paramsMust(t, b.AppendCString("p"))
		e := d.DisplayInto(b, 2, false, descr)
		want := "p  a_key (unsigned int)\n"
		if paramDescrsStreamSourceDebug {
			want = "p  a_key"
			if !errors.Is(e, ErrParamDescrsPrecondition) {
				t.Fatal("source assertion position", e)
			}
		} else if descr {
			want = "p  a_key (unsigned int) "
			if !errors.Is(e, ErrParamsStreamFailure) {
				t.Fatal("source null CString failure", e)
			}
		} else {
			paramsMust(t, e)
		}
		if got := pdStreamText(t, b); got != want {
			t.Fatal(descr, got, want)
		}
	}
}
func TestParamDescrsDisplayIntoBudgetsScratchAndHugeIndentPrefix(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	db := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, db)
	pdinsert(t, d, "name", ParamUInt, pdstr("desc"), nil, nil)
	b := pdStreamBuffer(t, nil)
	paramsMust(t, b.AppendCString("prefix"))
	baseline := db.Used()
	db.FailAfter(0)
	if e := d.DisplayInto(b, 0, false, true); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if db.Used() != baseline || b.Position() != 6 {
		t.Fatal("sort scratch failure root/prefix", db.Used(), baseline, b.Position())
	}
	db.FailAfter(-1)
	bb := arena.NewBudget(^uint64(0))
	out := pdStreamBuffer(t, bb)
	paramsMust(t, out.AppendByte('p'))
	bb.FailAfter(0)
	e := d.DisplayInto(out, ^uint32(0), false, true)
	if !errors.Is(e, arena.ErrInjectedFailure) || out.Position() != 64 {
		t.Fatal("full uint32 source indentation domain", e, out.Position())
	}
	for i := uint32(1); i < 64; i++ {
		v, e := out.Read(uint64(i))
		paramsMust(t, e)
		if v != ' ' {
			t.Fatal(i, v)
		}
	}
	if db.Used() != baseline {
		t.Fatal("source temporary names leak")
	}
	st := out.Stats()
	if bb.Used() != st.OwnerBytes+st.OverflowBytes || st.OwnerBytes != uint64(reflect.TypeOf(StringBuffer64{}).Size()) {
		t.Fatal("actual dst prefix storage", bb.Used(), st)
	}
	bb.FailAfter(-1)
}
func TestParamDescrsDisplayIntoGuardTimingAndIndependentOwner(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	b := pdStreamBuffer(t, nil)
	paramsMust(t, b.AppendCString("p"))
	paramsMust(t, d.Insert(Symbol{}, ParamUInt, pdstr("desc"), nil, nil))
	if e := d.DisplayInto(b, 2, false, false); !errors.Is(e, ErrParamDescrsPrecondition) {
		t.Fatal(e)
	}
	if got := pdStreamText(t, b); got != "p  " {
		t.Fatal("guard must occur at source name action", got)
	}
	if e := d.DisplayInto(nil, 0, false, false); !errors.Is(e, ErrStringBufferOwner) {
		t.Fatal(e)
	}
	copied := &ParamDescrs{Self: d, manager: m}
	if e := copied.DisplayInto(b, 0, false, false); !errors.Is(e, ErrParamDescrsPrecondition) {
		t.Fatal(e)
	}
	paramsMust(t, d.Close())
	if e := d.DisplayInto(b, 0, false, false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	paramsMust(t, b.Close())
	if e := d.DisplayInto(b, 0, false, false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestParamDescrsDisplayIntoSourceScratchCapacityWidths(t *testing.T) {
	for _, x := range []struct {
		old, width, next uint32
		overflow         bool
	}{
		{0, 4, 2, false}, {0, 8, 2, false}, {2, 8, 3, false}, {3, 8, 5, false}, {5, 4, 8, false},
		{357913940, 8, 536870910, false}, {357913941, 8, 0, true},
		{715827880, 4, 1073741820, false}, {715827881, 4, 0, true},
		{1431655765, 4, 0, true}, {^uint32(0), 8, 0, true},
	} {
		got, e := pdStreamNameCapacity(x.old, x.width)
		if x.overflow {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("source capacity byte overflow", x, got, e)
			}
		} else if e != nil || got != x.next {
			t.Fatal(x, got, e)
		}
	}
}
