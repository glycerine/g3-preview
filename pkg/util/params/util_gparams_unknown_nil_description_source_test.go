package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestGlobalParamsUnknownParameterLegalListOmitsNullDescriptions(t *testing.T) {
	ctx, symbols, _ := paramsTestOwner(t)
	budget := arena.NewBudget(^uint64(0))
	cfg := gparamsTestConfig(budget)
	cfg.Register = func(r GlobalParamsRegistrar) error {
		global, err := NewParamDescrs(symbols, ParamDescrsConfig{Budget: budget})
		if err != nil {
			return err
		}
		defer global.Close()
		if err = global.InsertName("known", ParamBool, nil, nil, nil); err != nil {
			return err
		}
		if err = r.RegisterGlobal(global); err != nil {
			return err
		}
		module, err := NewParamDescrs(symbols, ParamDescrsConfig{Budget: budget})
		if err != nil {
			return err
		}
		if err = module.InsertName("seed", ParamUInt, nil, nil, nil); err != nil {
			_ = module.Close()
			return err
		}
		if err = r.RegisterModule("sat", module); err != nil {
			_ = module.Close()
			return err
		}
		return nil
	}
	g, err := NewGlobalParams(symbols, ctx, cfg)
	paramsMust(t, err)
	t.Cleanup(func() {
		paramsMust(t, g.Close())
		if budget.Used() != 0 {
			t.Fatal("global descriptor owners", budget.Used())
		}
	})
	for _, tc := range []struct{ name, want string }{
		{"missing", "unknown parameter 'missing'\nLegal parameters are:\n  known (bool)\n"},
		{"sat.missing", "unknown parameter 'missing' at module 'sat'\nLegal parameters are:\n  seed (unsigned int)\n"},
	} {
		for _, action := range []func() error{
			func() error { return g.SetName(tc.name, "true") },
			func() error { _, e := g.GetValueName(tc.name); return e },
		} {
			err := action()
			if paramDescrsStreamSourceDebug {
				if !errors.Is(err, ErrParamDescrsPrecondition) {
					t.Fatal(tc.name, "checked legal-list assertion", err)
				}
			} else if err == nil || err.Error() != tc.want {
				t.Fatal(tc.name, "release legal-list diagnostic", err)
			}
		}
	}
}
