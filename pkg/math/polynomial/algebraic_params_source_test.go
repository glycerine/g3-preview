package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"testing"
)

func TestAlgebraicParametersAllGlobalFallbackLocalPrecedenceAndSnapshot(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "local"}[local], func(t *testing.T) {
			cfg, context, _ := algebraicManagerFixture(t)
			module, e := cfg.Globals.Symbols.Intern("algebraic")
			serviceMust(t, e)
			globals, e := cfg.Globals.Current.GetParams(module)
			serviceMust(t, e)
			// Distinct values detect swapped getter names, wrong defaults and accidental
			// reuse of another factorization option as well as wrong fallback order.
			for key, value := range map[string]uint32{"zero_accuracy": 7, "min_mag": 11, "factor_max_prime": 23, "factor_num_primes": 3, "factor_search_size": 101} {
				name := key
				serviceMust(t, context.SetUIntName(globals, &name, value))
			}
			factor := "factor"
			serviceMust(t, context.SetBoolName(globals, &factor, false))
			if local {
				for key, value := range map[string]uint32{"zero_accuracy": 13, "min_mag": 17, "factor_max_prime": 29, "factor_num_primes": 5, "factor_search_size": 103} {
					name := key
					serviceMust(t, context.SetUIntName(cfg.Params, &name, value))
				}
				serviceMust(t, context.SetBoolName(cfg.Params, &factor, true))
			}
			m, e := NewAlgebraicManager(cfg)
			serviceMust(t, e)
			defer func() { serviceMust(t, m.Close()) }()
			magnitude, accuracy, prime, trials, search := int32(-11), int32(-7), uint32(23), uint32(3), uint32(101)
			if local {
				magnitude, accuracy, prime, trials, search = -17, -13, 29, 5, 103
			}
			if m.minMagnitude != magnitude || m.zeroAccuracy != accuracy || m.factor != local || m.factorParams.MaxPrime != prime || m.factorParams.PrimeTrials != trials || m.factorParams.MaxSearchSize != search {
				t.Fatal("source six-getter binding", m.minMagnitude, m.zeroAccuracy, m.factor, m.factorParams)
			}
			if m.params.Helpers != nil || m.params.DestructorCause != nil {
				t.Fatal("generated helper reference survived constructor")
			}
			// Subsequent caller mutations cannot turn the constructor's captured values
			// into a live view, even through the source's empty public update method.
			for _, key := range []string{"zero_accuracy", "min_mag", "factor_max_prime", "factor_num_primes", "factor_search_size"} {
				name := key
				serviceMust(t, context.SetUIntName(globals, &name, 37))
				serviceMust(t, context.SetUIntName(cfg.Params, &name, 41))
			}
			serviceMust(t, context.SetBoolName(globals, &factor, !local))
			serviceMust(t, context.SetBoolName(cfg.Params, &factor, !local))
			serviceMust(t, m.UpdateParams(cfg.Params))
			if m.minMagnitude != magnitude || m.zeroAccuracy != accuracy || m.factor != local || m.factorParams.MaxPrime != prime || m.factorParams.PrimeTrials != trials || m.factorParams.MaxSearchSize != search {
				t.Fatal("constructor snapshot changed after helper destruction")
			}
		})
	}
}

func TestAlgebraicParameterDescriptorSchemaSourceOrder(t *testing.T) {
	cfg, _, budget := algebraicManagerFixture(t)
	d, e := uparams.NewParamDescrs(cfg.Globals.Symbols, uparams.ParamDescrsConfig{Budget: budget, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: budget, PageSize: 2}}})
	serviceMust(t, e)
	defer func() { serviceMust(t, d.Close()) }()
	serviceMust(t, CollectAlgebraicParamDescriptions(d))
	// Frozen source schema; this test invokes the production collector.
	rows := []struct {
		key, defaultValue, description string
		kind                           uparams.ParamKind
	}{
		{"zero_accuracy", "0", "one of the most time-consuming operations in the real algebraic number module is determining the sign of a polynomial evaluated at a sample point with non-rational algebraic number values. Let k be the value of this option. If k is 0, Z3 uses precise computation. Otherwise, the result of a polynomial evaluation is considered to be 0 if Z3 can show it is inside the interval (-1/2^k, 1/2^k)", uparams.ParamUInt},
		{"min_mag", "16", "Z3 represents algebraic numbers using a (square-free) polynomial p and an isolating interval (which contains one and only one root of p). This interval may be refined during the computations. This parameter specifies whether to cache the value of a refined interval or not. It says the minimal size of an interval for caching purposes is 1/2^16", uparams.ParamUInt},
		{"factor", "true", "use polynomial factorization to simplify polynomials representing algebraic numbers", uparams.ParamBool},
		{"factor_max_prime", "31", "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter limits the maximum prime number p to be used in the first step", uparams.ParamUInt},
		{"factor_num_primes", "1", "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. The search space may be reduced by factoring the polynomial in different GF(p)'s. This parameter specify the maximum number of finite factorizations to be considered, before lifiting and searching", uparams.ParamUInt},
		{"factor_search_size", "5000", "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter can be used to limit the search space", uparams.ParamUInt},
	}
	n, e := d.Size()
	serviceMust(t, e)
	if int(n) != len(rows) {
		t.Fatal("source schema size", n)
	}
	for i, row := range rows {
		name, e := d.ParamName(uint32(i))
		serviceMust(t, e)
		expected, e := cfg.Globals.Symbols.Intern(row.key)
		serviceMust(t, e)
		kind, e := d.KindName(row.key)
		serviceMust(t, e)
		value, present, e := d.DefaultName(row.key)
		serviceMust(t, e)
		module, hasModule, e := d.ModuleName(row.key)
		serviceMust(t, e)
		description, hasDescription, e := d.DescriptionName(row.key)
		serviceMust(t, e)
		if name != expected || kind != row.kind || !present || value != row.defaultValue || !hasModule || module != "algebraic" || !hasDescription || description != row.description {
			t.Fatal("source descriptor binding", i, row, value, module, description)
		}
	}
}
