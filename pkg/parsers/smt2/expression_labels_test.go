package smt2

// Source differential: src/parsers/smt2/smt2parser.cpp consume_attributes
// :lblpos/:lblneg branches.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2ExpressionLabelsPreserveAttributeNesting(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	var output bytes.Buffer
	script := `(set-logic QF_UF)(declare-const a Bool)
(assert (! a :lblpos first :lblneg |second label|))(check-sat)(labels)
(push)(assert (not a))(check-sat)(pop)(check-sat)(labels)`
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if output.String() != "sat\n(labels)\nsat\nsat\n(labels)\n" {
		t.Fatal("expression label results", output.String())
	}
	assertions, err := context.Assertions()
	if err != nil || len(assertions) != 1 {
		t.Fatal("labeled assertion", len(assertions), err)
	}
	m, err := context.Manager()
	if err != nil {
		t.Fatal(err)
	}
	outerNames := make([]string, 0, 1)
	innerNames := make([]string, 0, 1)
	current := assertions[0]
	for index, names := range []*[]string{&outerNames, &innerNames} {
		var positive bool
		var symbols []usymbol.Symbol
		matched, err := m.IsLabel(current, &positive, &symbols)
		if err != nil || !matched || len(symbols) != 1 {
			t.Fatal("label nesting", index, matched, symbols, err)
		}
		if positive != (index == 1) {
			t.Fatal("label polarity", index, positive)
		}
		text, err := m.Symbols.String(symbols[0])
		if err != nil {
			t.Fatal(err)
		}
		*names = append(*names, text)
		current, err = m.GetArg(current, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	if outerNames[0] != "second label" || innerNames[0] != "first" {
		t.Fatal("label names", outerNames, innerNames)
	}
}

func TestSMT2ExpressionLabelRequiresBooleanBodyAndSymbol(t *testing.T) {
	for name, script := range map[string]string{
		"non-boolean": `(assert (! 1 :lblpos tag))`,
		"non-symbol":  `(assert (! true :lblpos 1))`,
	} {
		t.Run(name, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Error(err)
				}
			})
			var output bytes.Buffer
			if err = ParseCommands(context, strings.NewReader(script), &output); err == nil {
				t.Fatal("invalid label accepted", script)
			}
		})
	}
}
