package api

// Typed public solver parameters cover the value categories used by the
// retained Z3-compatible SMT catalog.

import (
	"errors"
	"testing"
)

func TestSMTContextTypedParameterCatalog(t *testing.T) {
	c, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, c.Close()) })

	astTestMust(t, c.SetSMTBool(SMTArrayExtensional, false))
	astTestMust(t, c.SetSMTUInt(SMTRandomSeed, 73))
	astTestMust(t, c.SetSMTDouble(SMTRestartFactor, 1.25))
	astTestMust(t, c.SetSMTSymbol(SMTStringSolver, "seq"))
	astTestMust(t, c.SetSMTBool(SMTCoreExtendPatterns, true))
	astTestMust(t, c.SetSMTUInt(SMTStringRegexLengthAttempts, 17))

	for _, expected := range []struct {
		name, value string
	}{
		{"smt.array.extensional", "false"},
		{"smt.random_seed", "73"},
		{"smt.restart_factor", "1.25"},
		{"smt.string_solver", "seq"},
		{"smt.core.extend_patterns", "true"},
		{"smt.str.regex_automata_length_attempt_threshold", "17"},
	} {
		value, ok, err := c.command.GetOption(expected.name)
		astTestMust(t, err)
		if !ok || value != expected.value {
			t.Fatalf("%s = %q, %v", expected.name, value, ok)
		}
	}
}

func TestSMTContextTypedParametersValidateCatalogAndOwner(t *testing.T) {
	c, err := NewSMTContext()
	astTestMust(t, err)
	if err = c.SetSMTDouble(SMTDoubleParameter("smt.relevancy"), 1.5); err == nil {
		t.Fatal("double setter accepted an unsigned catalog entry")
	}
	if err = c.SetSMTBool(SMTBoolParameter("smt.not_a_parameter"), true); err == nil {
		t.Fatal("typed setter accepted an unknown catalog entry")
	}
	astTestMust(t, c.Close())
	if err = c.SetSMTUInt(SMTRandomSeed, 1); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed typed parameter setter", err)
	}
}
