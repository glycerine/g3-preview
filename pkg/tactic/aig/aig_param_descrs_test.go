package aig

import (
	"errors"
	"testing"

	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestAIGPassCollectParamDescrsSourceOrderAndMetadata(t *testing.T) {
	m, budget := aigFixture(t)
	d, err := uparams.NewParamDescrs(m.Symbols, uparams.ParamDescrsConfig{Budget: budget})
	aigMust(t, err)
	t.Cleanup(func() { aigMust(t, d.Close()) })
	pass := NewPass()
	aigMust(t, pass.CollectParamDescrs(d))
	aigMust(t, pass.CollectParamDescrs(d)) // source insertion keeps the first entry
	size, err := d.Size()
	aigMust(t, err)
	if size != 2 {
		t.Fatalf("descriptor count = %d, want 2", size)
	}
	for i, want := range []struct {
		name, description string
		kind              uparams.ParamKind
	}{
		{"max_memory", "(default: infty) maximum amount of memory in megabytes.", uparams.ParamUInt},
		{"aig_per_assertion", "(default: true) process one assertion at a time.", uparams.ParamBool},
	} {
		name, e := d.ParamName(uint32(i))
		aigMust(t, e)
		got, e := m.Symbols.String(name)
		aigMust(t, e)
		if got != want.name {
			t.Fatalf("descriptor %d = %q, want %q", i, got, want.name)
		}
		kind, e := d.Kind(name)
		aigMust(t, e)
		if kind != want.kind {
			t.Fatalf("%s kind = %v, want %v", got, kind, want.kind)
		}
		description, present, e := d.Description(name)
		aigMust(t, e)
		if !present || description != want.description {
			t.Fatalf("%s description = %q, present %v", got, description, present)
		}
		if _, present, e = d.Default(name); e != nil || present {
			t.Fatalf("%s unexpected separate default: %v, %v", got, present, e)
		}
	}
	if present, e := d.ContainsName("aig_default_gate_encoding"); e != nil || present {
		t.Fatalf("unadvertised encoding parameter: %v, %v", present, e)
	}
	if e := (*Pass)(nil).CollectParamDescrs(d); !errors.Is(e, ErrOwner) {
		t.Fatalf("nil pass: %v", e)
	}
	if e := pass.CollectParamDescrs(nil); !errors.Is(e, ErrOwner) {
		t.Fatalf("nil descriptors: %v", e)
	}
}
