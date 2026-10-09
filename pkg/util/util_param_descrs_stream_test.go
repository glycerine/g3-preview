package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func pdStreamBuffer(t *testing.T, b *arena.Budget) *StringBuffer64 {
	t.Helper()
	s, e := NewStringBuffer64(StringBufferConfig{})
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
func TestParamDescrsDisplayIntoGoBufferGrowth(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	db := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, db)
	pdinsert(t, d, "name", ParamUInt, pdstr("desc"), nil, nil)
	b := pdStreamBuffer(t, nil)
	paramsMust(t, b.AppendCString("prefix"))
	baseline := db.Used()
	out := pdStreamBuffer(t, nil)
	paramsMust(t, out.AppendByte('p'))
	const indent = uint32(4096)
	e := d.DisplayInto(out, indent, false, true)
	if e != nil || out.Position() <= uint64(indent) {
		t.Fatal("standard buffer growth", e, out.Position())
	}
	for i := uint32(1); i <= indent; i++ {
		v, e := out.Read(uint64(i))
		paramsMust(t, e)
		if v != ' ' {
			t.Fatal(i, v)
		}
	}
	if db.Used() != baseline {
		t.Fatal("descriptor arena usage changed")
	}
}
func TestParamDescrsDisplayIntoGuardTimingAndIndependentOwner(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	b := pdStreamBuffer(t, nil)
	paramsMust(t, b.AppendCString("p"))
	paramsMust(t, d.Insert(usymbol.Symbol{}, ParamUInt, pdstr("desc"), nil, nil))
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
