// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/math/polynomial/algebraic_params.pyg
func catalogCollectAlgebraicParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "zero_accuracy", util.ParamUInt, "one of the most time-consuming operations in the real algebraic number module is determining the sign of a polynomial evaluated at a sample point with non-rational algebraic number values. Let k be the value of this option. If k is 0, Z3 uses precise computation. Otherwise, the result of a polynomial evaluation is considered to be 0 if Z3 can show it is inside the interval (-1/2^k, 1/2^k)", "0", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min_mag", util.ParamUInt, "Z3 represents algebraic numbers using a (square-free) polynomial p and an isolating interval (which contains one and only one root of p). This interval may be refined during the computations. This parameter specifies whether to cache the value of a refined interval or not. It says the minimal size of an interval for caching purposes is 1/2^16", "16", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor", util.ParamBool, "use polynomial factorization to simplify polynomials representing algebraic numbers", "true", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_max_prime", util.ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter limits the maximum prime number p to be used in the first step", "31", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_num_primes", util.ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. The search space may be reduced by factoring the polynomial in different GF(p)'s. This parameter specify the maximum number of finite factorizations to be considered, before lifiting and searching", "1", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_search_size", util.ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter can be used to limit the search space", "5000", "algebraic"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectAlgebraicParams", catalogCollectAlgebraicParams)
}
