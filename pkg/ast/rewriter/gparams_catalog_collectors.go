// Parameter collectors remain with the C++ definitions they translate.
package rewriter

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/ast/rewriter/arith_rewriter_params.pyg
func catalogCollectArithRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "algebraic_number_evaluator", util.ParamBool, "simplify/evaluate expressions containing (algebraic) irrational numbers.", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mul_to_power", util.ParamBool, "collpase (* t ... t) into (^ t k), it is ignored if expand_power is true.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_power", util.ParamBool, "expand (^ t k) into (* t ... t) if  1 < k <= max_degree.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_tan", util.ParamBool, "replace (tan x) with (/ (sin x) (cos x)).", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_degree", util.ParamUInt, "max degree of algebraic numbers (and power operators) processed by simplifier.", "64", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sort_sums", util.ParamBool, "sort the arguments of + application.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "gcd_rounding", util.ParamBool, "use gcd rounding on integer arithmetic atoms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_lhs", util.ParamBool, "all monomials are moved to the left-hand-side, and the right-hand-side is just a constant.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_ineq_lhs", util.ParamBool, "rewrite inequalities so that right-hand-side is a constant.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_to_real", util.ParamBool, "eliminate to_real from arithmetic predicates that contain only integers.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_to_real", util.ParamBool, "distribute to_real over * and +.", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "eq2ineq", util.ParamBool, "expand equalities into two inequalities", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_rem", util.ParamBool, "replace (rem x y) with (ite (>= y 0) (mod x y) (- (mod x y))).", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/array_rewriter_params.pyg
func catalogCollectArrayRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "expand_select_store", util.ParamBool, "replace a (select (store ...) ...) term by an if-then-else term", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_store_eq", util.ParamBool, "reduce (store ...) = (store ...) with a common base into selects", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sort_store", util.ParamBool, "sort nested stores when the indices are known to be different", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/bool_rewriter_params.pyg
func catalogCollectBoolRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "ite_extra_rules", util.ParamBool, "extra ite simplifications, these additional simplifications may reduce size locally but increase globally", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", util.ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_and", util.ParamBool, "conjunctions are rewritten using negation and disjunctions", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "local_ctx", util.ParamBool, "perform local (i.e., cheap) context simplifications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "local_ctx_limit", util.ParamUInt, "limit for applying local context simplifier", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_distinct", util.ParamBool, "expand a distinct predicate into a quadratic number of disequalities", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_distinct_threshold", util.ParamUInt, "when blast_distinct is true, only distinct expressions with less than this number of arguments are blasted", "4294967295", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/bv_rewriter_params.pyg
func catalogCollectBvRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "udiv2mul", util.ParamBool, "convert constant udiv to mul", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "split_concat_eq", util.ParamBool, "split equalities of the form (= (concat t1 t2) t3)", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bit2bool", util.ParamBool, "try to convert bit-vector terms of size 1 into Boolean terms", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_eq_value", util.ParamBool, "blast (some) Bit-vector equalities into bits", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_sign_ext", util.ParamBool, "expand sign-ext operator using concat and extract", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hi_div0", util.ParamBool, "use the 'hardware interpretation' for division by zero (for bit-vector terms)", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mul2concat", util.ParamBool, "replace multiplication by a power of two into a concatenation", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bvnot2arith", util.ParamBool, "replace (bvnot x) with (bvsub -1 x)", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_sort_ac", util.ParamBool, "sort the arguments of all AC operators", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_trailing", util.ParamBool, "lean removal of trailing zeros", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_extract_prop", util.ParamBool, "attempt to partially propagate extraction inwards", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_not_simpl", util.ParamBool, "apply simplifications for bvnot", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_ite2id", util.ParamBool, "rewrite ite that can be simplified to identity", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_le_extra", util.ParamBool, "additional bu_(u/s)le simplifications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_urem_simpl", util.ParamBool, "additional simplification for bvurem", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/fpa_rewriter_params.pyg
func catalogCollectFpaRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "hi_fp_unspecified", util.ParamBool, "use the 'hardware interpretation' for unspecified values in fp.to_ubv, fp.to_sbv, fp.to_real, and fp.to_ieee_bv", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/poly_rewriter_params.pyg
func catalogCollectPolyRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "som", util.ParamBool, "put polynomials in som-of-monomials form", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "som_blowup", util.ParamUInt, "maximum number of monomials generated when putting a polynomial in sum-of-monomials normal form", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hoist_mul", util.ParamBool, "hoist multiplication over summation to minimize number of multiplications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hoist_cmul", util.ParamBool, "hoist constant multiplication over summation to minimize number of multiplications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", util.ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/rewriter_params.pyg
func catalogCollectRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_steps", util.ParamUInt, "maximum number of steps", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", util.ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_ite_arith", util.ParamBool, "push if-then-else over arithmetic terms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_ite_bv", util.ParamBool, "push if-then-else over bit-vector terms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_cheap_ite", util.ParamBool, "pull if-then-else terms when cheap.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_ineq_consistency_test_max", util.ParamUInt, "max size of conjunctions on which to perform consistency test based on inequalities on bitvectors.", "0", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "cache_all", util.ParamBool, "cache all intermediate results.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rewrite_patterns", util.ParamBool, "rewrite patterns.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_patterns_on_ground_qbody", util.ParamBool, "ignores patterns on quantifiers that don't mention their bound variables.", "true", "rewriter"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectArithRewriterParams", catalogCollectArithRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectArrayRewriterParams", catalogCollectArrayRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectBoolRewriterParams", catalogCollectBoolRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectBvRewriterParams", catalogCollectBvRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectFpaRewriterParams", catalogCollectFpaRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectPolyRewriterParams", catalogCollectPolyRewriterParams)
	util.RegisterGlobalParamsCollector("catalogCollectRewriterParams", catalogCollectRewriterParams)
}
