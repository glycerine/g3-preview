package smt2

// Source: src/cmd_context/simplify_cmd.cpp command-local parameter parsing and
// default simplified-term output.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/glycerine/g3/pkg/ast/rewriter"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func (p *commandParser) simplifyCommand() (resultErr error) {
	expression, _, err := p.expression()
	if err != nil {
		return err
	}
	globals, err := p.context.GlobalParameters()
	if err != nil {
		return err
	}
	settings, err := globals.Context.NewValue()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, globals.Context.Del(&settings)) }()
	m, err := p.astManager()
	if err != nil {
		return err
	}
	descriptions, err := uparams.NewParamDescrs(m.Symbols, uparams.ParamDescrsConfig{Budget: m.Budget()})
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, descriptions.Close()) }()
	if err = rewriter.TheoryRewriterParameterDescriptions(descriptions); err != nil {
		return err
	}
	printResult := true
	for p.token != RightParenToken && p.token != EOFToken {
		if p.token != KeywordToken {
			return fmt.Errorf("invalid simplify command, keyword expected")
		}
		key := strings.TrimPrefix(p.scanner.ID(), ":")
		key = uparams.NormParamName(&key)
		if err = p.advance(); err != nil {
			return err
		}
		if key == "print" {
			if p.token != SymbolToken || p.scanner.ID() != "true" && p.scanner.ID() != "false" {
				return fmt.Errorf("invalid simplify parameter value, true or false expected")
			}
			printResult = p.scanner.ID() == "true"
			if err = p.advance(); err != nil {
				return err
			}
			continue
		}
		symbol, e := m.Symbols.Intern(key)
		if e != nil {
			return e
		}
		kind, e := descriptions.KindInModule(&symbol)
		if e != nil {
			return e
		}
		if kind == uparams.ParamInvalid {
			return fmt.Errorf("invalid simplify command, unknown parameter %s", key)
		}
		key, e = m.Symbols.String(symbol)
		if e != nil {
			return e
		}
		if err = p.tacticParameter(globals.Context, &settings, key, kind); err != nil {
			return err
		}
	}
	if err = p.requireEnd(); err != nil {
		return err
	}
	somName := "som"
	som, err := globals.Context.GetBoolName(&settings, &somName, false)
	if err != nil {
		return err
	}
	if som {
		flatName := "flat"
		if err = globals.Context.SetBoolName(&settings, &flatName, true); err != nil {
			return err
		}
	}
	result, err := p.context.Simplify(expression, &settings)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, m.DelRef(&result)) }()
	if !printResult {
		return nil
	}
	simplified, err := m.RefGet(result)
	if err != nil {
		return err
	}
	if err = p.context.WriteExpression(p.out, simplified); err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.out)
	return err
}
