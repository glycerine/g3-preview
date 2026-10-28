package smtcore

// Source: src/smt/params/theory_pb_params.h:25-39;
// src/smt/params/theory_pb_params.cpp:22-37; src/smt/params/smt_params_helper.pyg:55-58.

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSMTPBConstructorDoesNotUpdateAndLiteralDefaults(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	v, e := NewSMTPBParams(b)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, v.Close())
		if b.Used() != 0 {
			t.Fatal("real outer refund")
		}
	}()
	got, e := v.State()
	paramsMust(t, e)
	want := SMTPBState{1000, true, true, false}
	if got != want {
		t.Fatal("source four initializer defaults, constructor does not update", got)
	}
	if v.Helpers != nil || v.Context != nil || v.Globals != nil {
		t.Fatal("erased parameter/helper reads")
	}
}

func pbNamedColdGraph(t *testing.T, v *SMTPBParams, g *uparams.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		n += gparamsGraph(g.Current)
	}
	for x := g.Failed; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	for x := g.Superseded; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	if v != nil {
		if v.self != v || v.Budget != b {
			t.Fatal("actual wrapper")
		}
		n += v.header
		for f := v.Helpers; f != nil; f = f.Next {
			if f.Self != f || f.Owner != &v.ModuleParamsOwner {
				t.Fatal("actual lexical helper")
			}
			n += f.Header
		}
	}
	if n != b.Used() {
		t.Fatal("immediate once-only graph", n, b.Used())
	}
}
func TestSMTPBFourGettersIndependentLocalModuleWrongKindLiteral(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPBParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	module, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.Current.GetParams(module)
	paramsMust(t, e)
	cases := []struct {
		key, field  string
		isBool, def bool
		u           uint32
	}{{"pb.conflict_frequency", "ConflictFrequency", false, false, 1000}, {"pb.learn_complements", "LearnComplements", true, true, 0}, {"pb.enable_compilation", "EnableCompilation", true, true, 0}, {"pb.enable_simplex", "EnableSimplex", true, false, 0}}
	for _, tc := range cases {
		for mode := 0; mode < 4; mode++ {
			paramsMust(t, c.Reset(p))
			paramsMust(t, c.Reset(global))
			wantB, wantU := tc.def, tc.u
			if mode < 3 {
				if tc.isBool {
					wantB = !tc.def
					paramsMust(t, c.SetBoolName(global, &tc.key, wantB))
				} else {
					wantU = 77
					paramsMust(t, c.SetUIntName(global, &tc.key, wantU))
				}
			}
			if mode == 1 {
				if tc.isBool {
					wantB = tc.def
					paramsMust(t, c.SetBoolName(p, &tc.key, wantB))
				} else {
					wantU = 13
					paramsMust(t, c.SetUIntName(p, &tc.key, wantU))
				}
			}
			if mode == 2 {
				if tc.isBool {
					paramsMust(t, c.SetUIntName(p, &tc.key, 99))
				} else {
					paramsMust(t, c.SetBoolName(p, &tc.key, true))
				}
			}
			paramsMust(t, v.Update(c, p, g))
			got, e := v.State()
			paramsMust(t, e)
			field := testFieldBySourceName(reflect.ValueOf(got), tc.field)
			if tc.isBool {
				if field.Bool() != wantB {
					t.Fatal(tc, mode, got)
				}
			} else if field.Uint() != uint64(wantU) {
				t.Fatal(tc, mode, got)
			}
		}
	}
	pbNamedColdGraph(t, v, g, b)
}
func TestSMTPBEveryConstructorAndWarmHelperReservation(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(ordinal)
		v, cause := NewSMTPBParams(b)
		if ordinal == 0 {
			if v != nil || !errors.Is(cause, arena.ErrInjectedFailure) || b.Used() != 0 {
				t.Fatal("header refusal", v, cause)
			}
		} else {
			paramsMust(t, cause)
			if b.Used() != v.header || v.Helpers != nil {
				t.Fatal("only actual wrapper")
			}
			paramsMust(t, v.Close())
		}
	}
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		_, e := g.Symbols.Intern("smt")
		paramsMust(t, e)
		v, e := NewSMTPBParams(b)
		paramsMust(t, e)
		key := "pb.conflict_frequency"
		paramsMust(t, c.SetUIntName(p, &key, 9))
		b.FailAfter(ordinal)
		cause := v.Update(c, p, g)
		pbNamedColdGraph(t, v, g, b)
		if ordinal == 0 {
			if !errors.Is(cause, arena.ErrInjectedFailure) || v.state.ConflictFrequency != 1000 || v.Helpers != nil {
				t.Fatal("literal helper before first RHS", cause)
			}
		} else {
			paramsMust(t, cause)
			if v.state.ConflictFrequency != 9 {
				t.Fatal("published first RHS")
			}
		}
		b.FailAfter(-1)
		paramsMust(t, v.Close())
		pbNamedColdGraph(t, nil, g, b)
	}
}

const pbDefaultDisplay = "m_pb_conflict_frequency=1000\nm_pb_learn_complements=1\nm_pb_enable_compilation=1\nm_pb_enable_simplex=0\n"

func TestSMTPBFourDisplayChainsGoBufferShowPosAndFileBadbit(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	v, e := NewSMTPBParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := v.Display(out)
		got, e := out.String()
		paramsMust(t, e)
		if len(got) > len(pbDefaultDisplay) || got != pbDefaultDisplay[:len(got)] {
			t.Fatal("literal independent chain prefix", ordinal, got, cause)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		if cause == nil && got != pbDefaultDisplay {
			t.Fatal("complete display", got)
		}
		stats := out.Stats()
		if sb.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
			t.Fatal("actual output cold graph before fault lift", ordinal, sb.Used(), stats)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("owned output once")
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 0 {
		t.Fatal("Go buffer output did not complete", first)
	}
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	if got != "m_pb_conflict_frequency=1000\nm_pb_learn_complements=+1\nm_pb_enable_compilation=+1\nm_pb_enable_simplex=+0\n" {
		t.Fatal("signed bool with unsigned frequency", got)
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "pb-output-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("nonthrowing file badbit")
	}
	paramsMust(t, out.Close())
	if b.Used() != v.header {
		t.Fatal("only actual parent remains")
	}
}
func TestSMTPBEveryUnwarmedSharedModuleAcquisitionReservation(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		bc := arena.NewBudget(^uint64(0))
		c, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := uparams.NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		local, e := c.NewValue()
		paramsMust(t, e)
		v, e := NewSMTPBParams(b)
		paramsMust(t, e)
		b.FailAfter(ordinal)
		cause := v.Update(c, &local, g)
		n := g.OwnerBytes + catalogGraph(catalog) + gparamsGraph(g.Current) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if v != nil {
			n += v.header
			for f := v.Helpers; f != nil; f = f.Next {
				n += f.Header
			}
		}
		if n != b.Used() {
			t.Fatal("exact shared graph before lift", ordinal, n, b.Used())
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("source constructor actual leaf", cause)
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		paramsMust(t, c.Del(&local))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bc.Used() != 0 {
			t.Fatal("all named refund once", ordinal, b.Used(), bc.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	// Symbol spellings use Go storage; count only remaining arena acquisitions.
	if first != 1 {
		t.Fatal("actual shared symbol/module acquisition absent", first)
	}
	t.Log("first successful unwarmed PB update", first)
}

func TestSMTPBNullGlobalReachedHelperAndActualOwnerGuards(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTPBParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	paramsMust(t, g.FinalizeSource())
	before := v.state
	cause := v.Update(c, p, g)
	if uparams.GlobalParamsSourceDebug {
		if !errors.Is(cause, uparams.ErrGlobalParamsAssertion) {
			t.Fatal("source namespace assertion", cause)
		}
	} else if !errors.Is(cause, uparams.ErrGlobalParamsUninitialized) {
		t.Fatal("guarded null imp", cause)
	}
	if v.state != before || v.Helpers != nil || v.Serial != 1 {
		t.Fatal("actual failed helper before first RHS", v.state, v.Serial)
	}
	pbNamedColdGraph(t, v, g, b)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	if got != pbDefaultDisplay {
		t.Fatal("source constructor complete independently of failed later update", got)
	}
	paramsMust(t, out.Close())
	copyOwner := *v
	if e = copyOwner.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("accidental owner copy before mutation", e)
	}
	paramsMust(t, g.InitializeSource())
	paramsMust(t, v.Update(c, p, g))
	if v.state != before || v.Serial != 2 {
		t.Fatal("actual completed-object retry after global restoration")
	}
	pbNamedColdGraph(t, v, g, b)
	paramsMust(t, v.Close())
	if e = v.Update(c, p, g); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("ended lifetime", e)
	}
	pbNamedColdGraph(t, nil, g, b)
}
