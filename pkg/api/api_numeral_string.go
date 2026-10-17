// Source: src/api/api_numeral.cpp:187-228.
package api

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

var errAPINumeralArgument = errors.New("api: invalid numeral argument")

// numeralString is the retained arithmetic and FPA portion of
// Z3_get_numeral_string. The Go string owns its bytes after the source's
// scoped MPF and temporary formatting context have been released.
func (c *apiContextResults) numeralString(h ast.ASTHandle) (string, error) {
	if err := c.open(); err != nil {
		return "", err
	}
	value, _, matched, err := c.manager.ArithmeticNumeralText(h)
	if err != nil || matched {
		return value, err
	}
	value, matched, err = c.manager.FPANumeralString(h)
	if err != nil || matched {
		return value, err
	}
	return "", errAPINumeralArgument
}
