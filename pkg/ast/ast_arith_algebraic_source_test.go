package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTArithmeticAlgebraicExternalTranslationOwnsDistinctValue(t *testing.T) {
	sy, q, budget := astTestOwners(t)
	context, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: budget, PageEntries: 2})
	astTestMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(budget))
	astTestMust(t, e)
	cfg := gparamsTestConfig(budget)
	cfg.Catalog = catalog
	globals, e := uparams.NewGlobalParamsService(sy, context, cfg)
	astTestMust(t, e)
	astTestMust(t, globals.InitializeSource())
	previous, e := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, e)
	defer func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	}()

	newManager := func() *ASTManager {
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		return m
	}
	source, target := newManager(), newManager()
	defer func() {
		astTestMust(t, source.Close())
		astTestMust(t, target.Close())
	}()
	sw, e := source.arithmeticPlugin.algebraicWrapper()
	astTestMust(t, e)

	n := unumeral.NewInt(2)
	var two, root polynomial.Algebraic
	astTestMust(t, sw.manager.SetInteger(&two, &n))
	astTestMust(t, sw.manager.Root(&two, 2, &root))
	astTestMust(t, sw.manager.Del(&two))
	id, e := sw.mkID(sw.manager.AlgebraicRootCore, &root)
	astTestMust(t, e)
	astTestMust(t, sw.manager.Del(&root))
	parameter, e := source.allocParameter()
	astTestMust(t, e)
	source.parameterAt(parameter.slot).Kind = ASTParameterExternal
	source.parameterAt(parameter.slot).external = id

	translated, e := source.TranslateArithmeticExternalParameter(parameter, target)
	astTestMust(t, e)
	tw := target.arithmeticPlugin.algebraic
	if tw == nil || tw == sw || tw.manager == sw.manager {
		t.Fatal("translation did not construct a distinct target algebraic owner")
	}
	targetRecord, e := target.parameterOfKind(translated, ASTParameterExternal)
	astTestMust(t, e)
	targetValue, e := tw.value(targetRecord.external)
	astTestMust(t, e)
	var sourceText, targetText bytes.Buffer
	astTestMust(t, sw.manager.DisplayDecimal(&sourceText, &sw.values[id], 12))
	astTestMust(t, tw.manager.DisplayDecimal(&targetText, targetValue, 12))
	if sourceText.String() != targetText.String() {
		t.Fatal("cross-manager algebraic value changed", sourceText.String(), targetText.String())
	}

	astTestMust(t, sw.recycle(id))
	astTestMust(t, source.dropParameter(parameter))
	// Reusing the source ID cannot mutate or invalidate the copied target row.
	var againTwo, sourceAgain polynomial.Algebraic
	astTestMust(t, sw.manager.SetInteger(&againTwo, &n))
	astTestMust(t, sw.manager.Root(&againTwo, 2, &sourceAgain))
	astTestMust(t, sw.manager.Del(&againTwo))
	reused, e := sw.mkID(sw.manager.AlgebraicRootCore, &sourceAgain)
	astTestMust(t, e)
	astTestMust(t, sw.manager.Del(&sourceAgain))
	if reused != id {
		t.Fatal("source id_gen did not reuse the recycled ID", reused, id)
	}
	targetText.Reset()
	astTestMust(t, tw.manager.DisplayDecimal(&targetText, targetValue, 12))
	if targetText.Len() == 0 {
		t.Fatal("target copy was invalidated by source ID reuse")
	}

	astTestMust(t, sw.recycle(reused))
	astTestMust(t, tw.recycle(targetRecord.external))
	astTestMust(t, target.dropParameter(translated))
	if budget.Used() == 0 {
		t.Fatal("borrowed global services unexpectedly closed")
	}
}

func TestASTArithmeticAlgebraicNumeralDeletionRecyclesExternalID(t *testing.T) {
	sy, q, budget := astTestOwners(t)
	context, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: budget, PageEntries: 2})
	astTestMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(budget))
	astTestMust(t, e)
	cfg := gparamsTestConfig(budget)
	cfg.Catalog = catalog
	globals, e := uparams.NewGlobalParamsService(sy, context, cfg)
	astTestMust(t, e)
	astTestMust(t, globals.InitializeSource())
	previous, e := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, e)
	defer func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	}()

	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.Close()) }()
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	am, e := m.ArithmeticAlgebraicManager()
	astTestMust(t, e)
	n := unumeral.NewInt(2)
	var two, root polynomial.Algebraic
	astTestMust(t, am.SetInteger(&two, &n))
	astTestMust(t, am.Root(&two, 2, &root))
	astTestMust(t, am.Del(&two))
	h, e := m.ArithmeticAlgebraicNumeral(&root)
	astTestMust(t, e)
	astTestMust(t, am.Del(&root))
	w := m.arithmeticPlugin.algebraic
	if w.nextID != 1 || len(w.freeIDs) != 0 {
		t.Fatal("irrational declaration did not allocate its source ID", w.nextID, w.freeIDs)
	}
	astTestMust(t, m.IncRef(h))
	astTestMust(t, m.DecRef(h))
	if len(w.freeIDs) != 1 || w.freeIDs[0] != 0 {
		t.Fatal("attached arithmetic external DEL did not recycle ID", w.freeIDs)
	}
	var closeTwo, closeRoot polynomial.Algebraic
	astTestMust(t, am.SetInteger(&closeTwo, &n))
	astTestMust(t, am.Root(&closeTwo, 2, &closeRoot))
	astTestMust(t, am.Del(&closeTwo))
	_, e = m.ArithmeticAlgebraicNumeral(&closeRoot)
	astTestMust(t, e)
	astTestMust(t, am.Del(&closeRoot))
}

func TestASTArithmeticAlgebraicEqualityAndDistinctness(t *testing.T) {
	sy, q, budget := astTestOwners(t)
	context, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: budget, PageEntries: 2})
	astTestMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(sy, catalogTestConfig(budget))
	astTestMust(t, e)
	cfg := gparamsTestConfig(budget)
	cfg.Catalog = catalog
	globals, e := uparams.NewGlobalParamsService(sy, context, cfg)
	astTestMust(t, e)
	astTestMust(t, globals.InitializeSource())
	previous, e := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, e)
	defer func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	}()
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.Close()) }()
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	am, e := m.ArithmeticAlgebraicManager()
	astTestMust(t, e)

	makeRoot := func(n int32) polynomial.Algebraic {
		integer := unumeral.NewInt(n)
		var basic, root polynomial.Algebraic
		astTestMust(t, am.SetInteger(&basic, &integer))
		astTestMust(t, am.Root(&basic, 2, &root))
		astTestMust(t, am.Del(&basic))
		return root
	}
	two, anotherTwo, three := makeRoot(2), makeRoot(2), makeRoot(3)
	a, e := m.ArithmeticAlgebraicNumeral(&two)
	astTestMust(t, e)
	b, e := m.ArithmeticAlgebraicNumeral(&anotherTwo)
	astTestMust(t, e)
	c, e := m.ArithmeticAlgebraicNumeral(&three)
	astTestMust(t, e)
	astTestMust(t, am.Del(&two))
	astTestMust(t, am.Del(&anotherTwo))
	astTestMust(t, am.Del(&three))

	equal, e := m.AreEqual(a, b)
	astTestMust(t, e)
	distinctAB, e := m.AreDistinct(a, b)
	astTestMust(t, e)
	distinctAC, e := m.AreDistinct(a, c)
	astTestMust(t, e)
	if !equal || distinctAB || !distinctAC {
		t.Fatal("source irrational algebraic equality", equal, distinctAB, distinctAC)
	}
}
