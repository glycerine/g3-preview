// Source: src/ast/ast.cpp:1520-1561.
package ast

import (
	"errors"
	"testing"
)

func astTranslationManagers(t *testing.T) (*ASTManager, *ASTManager) {
	t.Helper()
	fromSymbols, fromNumerals, fromBudget := astTestOwners(t)
	toSymbols, toNumerals, toBudget := astTestOwners(t)
	from, err := NewASTManager(ASTManagerConfig{Symbols: fromSymbols, Numerals: fromNumerals, Budget: fromBudget})
	astTestMust(t, err)
	to, err := NewASTManager(ASTManagerConfig{Symbols: toSymbols, Numerals: toNumerals, Budget: toBudget})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, to.Close())
		astTestMust(t, from.Close())
	})
	return from, to
}

func TestASTCopyFamiliesPluginsPreservesOrderAndConcreteKinds(t *testing.T) {
	from, to := astTranslationManagers(t)
	datalog, err := from.RegisterDatalogPlugin()
	astTestMust(t, err)
	array, err := from.RegisterArrayPlugin()
	astTestMust(t, err)
	datatype, err := from.RegisterDatatypePlugin()
	astTestMust(t, err)
	arithmetic, err := from.RegisterArithmeticPlugin()
	astTestMust(t, err)
	if arithmetic != 5 || datalog != 6 || array != 7 || datatype != 8 {
		t.Fatal("source family order", arithmetic, datalog, array, datatype)
	}

	from.mu.Lock()
	custom, err := from.Symbols.Intern("custom-family")
	if err == nil {
		_, err = from.families.MkID(custom)
	}
	from.mu.Unlock()
	astTestMust(t, err)

	astTestMust(t, to.CopyFamiliesPlugins(from))
	for id := FamilyID(0); id < from.families.Stats().NextID; id++ {
		sourceName, e := from.families.GetName(id)
		astTestMust(t, e)
		text, e := from.Symbols.String(sourceName)
		astTestMust(t, e)
		targetName, e := to.Symbols.Intern(text)
		astTestMust(t, e)
		got, e := to.families.GetID(targetName)
		astTestMust(t, e)
		if got != id {
			t.Fatal("family ID", text, got, id)
		}
		sourcePlugin := from.pluginAt(id)
		targetPlugin := to.pluginAt(id)
		if sourcePlugin != targetPlugin {
			t.Fatal("plugin binding", id, sourcePlugin, targetPlugin)
		}
	}
	if to.pluginAt(9).present {
		t.Fatal("plugin synthesized for a family without one")
	}

	used := to.BudgetField.Used()
	arithOwner, datalogOwner := to.arithmeticPlugin, to.auxiliaryPlugin(astDatalogPlugin)
	astTestMust(t, to.CopyFamiliesPlugins(from))
	if to.BudgetField.Used() != used || to.arithmeticPlugin != arithOwner || to.auxiliaryPlugin(astDatalogPlugin) != datalogOwner {
		t.Fatal("copy must be idempotent")
	}
}

func TestASTCopyFamiliesPluginsRejectsDivergentFamilyOrder(t *testing.T) {
	from, to := astTranslationManagers(t)
	_, err := from.RegisterDatalogPlugin()
	astTestMust(t, err)
	_, err = to.RegisterArrayPlugin()
	astTestMust(t, err)
	if err = to.CopyFamiliesPlugins(from); !errors.Is(err, ErrASTDomain) {
		t.Fatal("divergent family order", err)
	}
}

func TestASTCopyFamiliesPluginsCopiesFormatPlugin(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	from, err := NewASTFormatManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	astTestMust(t, err)
	to, err := NewASTFormatManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, to.Close())
		astTestMust(t, from.Close())
	})
	id, err := from.GetFormatFamilyID()
	astTestMust(t, err)
	astTestMust(t, to.CopyFamiliesPlugins(from))
	if binding := to.pluginAt(id); !binding.present || binding.kind != astFormatPlugin || to.formatPlugin == nil {
		t.Fatal("format plugin was not copied", id, binding)
	}
}
