// Source: src/math/subpaving/tactic/expr2subpaving.cpp:90-148, 221-264, 332-363.
package tactic

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrExprConverterUnsupported = errors.New("subpaving tactic: arithmetic expression conversion is not yet implemented")

// HardwareExprConverter owns the source expr2var references created for
// arithmetic variables. Its AST manager and hardware engine are borrowed.
type HardwareExprConverter struct {
	self   *HardwareExprConverter
	ast    *ast.ASTManager
	engine *HardwareEngine
	family ast.FamilyID
	vars   map[ast.ASTHandle]uint32
	order  []ast.ASTHandle
	closed bool
}

func NewHardwareExprConverter(m *ast.ASTManager, engine *HardwareEngine) (*HardwareExprConverter, error) {
	if m == nil || engine == nil {
		return nil, arena.ErrInvalidSlot
	}
	if _, err := engine.Shell(); err != nil {
		return nil, err
	}
	family, err := m.ArithmeticFamilyID()
	if err != nil {
		return nil, err
	}
	c := &HardwareExprConverter{ast: m, engine: engine, family: family, vars: make(map[ast.ASTHandle]uint32)}
	c.self = c
	return c, nil
}

func (c *HardwareExprConverter) check() error {
	if c == nil || c.self != c {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	_, err := c.engine.Shell()
	return err
}

// InternalizeTerm implements numeral and variable leaves, to_real, and add.
// The caller owns numerator and denominator.
// math.MaxUint32 is the source null_var value for constant terms.
func (c *HardwareExprConverter) InternalizeTerm(t ast.ASTHandle, numerator, denominator *numeral.Int) (uint32, error) {
	return c.process(t, numerator, denominator)
}

func (c *HardwareExprConverter) process(t ast.ASTHandle, numerator, denominator *numeral.Int) (uint32, error) {
	if err := c.check(); err != nil {
		return 0, err
	}
	if numerator == nil || denominator == nil {
		return 0, arena.ErrInvalidSlot
	}
	kind, err := c.ast.GetKind(t)
	if err != nil {
		return 0, err
	}
	if kind != ast.ASTApp && kind != ast.ASTVar {
		return 0, ast.ErrASTDomain
	}
	sort, err := c.ast.GetSort(t)
	if err != nil {
		return 0, err
	}
	family, err := c.ast.GetFamilyID(sort)
	if err != nil {
		return 0, err
	}
	sortKind, err := c.ast.GetDeclKind(sort)
	if err != nil {
		return 0, err
	}
	if family != c.family || (sortKind != 0 && sortKind != 1) {
		return 0, ast.ErrASTDomain
	}
	if kind == ast.ASTApp {
		value, _, matched, err := c.ast.ArithmeticNumeralText(t)
		if err != nil {
			return 0, err
		}
		if matched {
			r := numeral.NewRational()
			defer c.engine.qm.Del(&r)
			if err := c.engine.qm.SetString(&r, value); err != nil {
				return 0, err
			}
			if err := c.engine.qm.Numerator(&r, numerator); err != nil {
				return 0, err
			}
			if err := c.engine.qm.Denominator(&r, denominator); err != nil {
				return 0, err
			}
			return math.MaxUint32, nil
		}
		declFamily, err := c.ast.GetFamilyID(t)
		if err != nil {
			return 0, err
		}
		if declFamily == c.family {
			operator, err := c.ast.GetDeclKind(t)
			if err != nil {
				return 0, err
			}
			switch operator {
			case 6: // OP_ADD
				return c.processAdd(t, numerator, denominator)
			case 14: // OP_TO_REAL
				arg, err := c.ast.GetArg(t, 0)
				if err != nil {
					return 0, err
				}
				return c.process(arg, numerator, denominator)
			default:
				return 0, ErrExprConverterUnsupported
			}
		}
	}
	if variable, ok := c.vars[t]; ok {
		if err := c.engine.qm.Z.SetInt32(numerator, 1); err != nil {
			return 0, err
		}
		return variable, c.engine.qm.Z.SetInt32(denominator, 1)
	}
	if err := c.ast.IncRef(t); err != nil {
		return 0, err
	}
	shell, err := c.engine.Shell()
	if err != nil {
		c.ast.DecRef(t)
		return 0, err
	}
	variable, err := shell.MkVar(sortKind == 1)
	if err != nil {
		c.ast.DecRef(t)
		return 0, err
	}
	c.vars[t] = variable
	c.order = append(c.order, t)
	if err := c.engine.qm.Z.SetInt32(numerator, 1); err != nil {
		return 0, err
	}
	return variable, c.engine.qm.Z.SetInt32(denominator, 1)
}

// processAdd uses the two live unsynch_mpz_manager buffer instantiations from
// src/util/scoped_numeral_buffer.h. It computes the common denominator before
// handing exact integer coefficients to the hardware wrapper.
func (c *HardwareExprConverter) processAdd(t ast.ASTHandle, n, d *numeral.Int) (variable uint32, err error) {
	qm := c.engine.qm
	z := qm.Z
	var ns, ds numeral.ScopedIntegerBuffer16
	if err = ns.Construct(z); err != nil {
		return 0, err
	}
	if err = ds.Construct(z); err != nil {
		_ = ns.Cleanup()
		return 0, err
	}
	constant, argConstant := numeral.NewRational(), numeral.NewRational()
	var argN, argD, factor, sumConstant numeral.Int
	defer func() {
		// A stopped buffer destructor retains its roots for explicit cleanup.
		for _, b := range []*numeral.ScopedIntegerBuffer16{&ds, &ns} {
			e := b.SourceClose()
			err = errors.Join(err, e)
			if err != nil {
				err = errors.Join(err, b.Cleanup())
			}
		}
		err = errors.Join(err, z.Del(&sumConstant), z.Del(&factor), z.Del(&argD), z.Del(&argN), qm.Del(&argConstant), qm.Del(&constant))
	}()
	count, err := c.ast.GetNumArgs(t)
	if err != nil {
		return 0, err
	}
	vars := make([]uint32, 0, count)
	for i := uint32(0); i < count; i++ {
		arg, e := c.ast.GetArg(t, i)
		if e != nil {
			return 0, e
		}
		x, e := c.process(arg, &argN, &argD)
		if e != nil {
			return 0, e
		}
		if x == math.MaxUint32 {
			if e = qm.SetParts(&argConstant, &argN, &argD); e != nil {
				return 0, e
			}
			if e = qm.Add(&constant, &argConstant, &constant); e != nil {
				return 0, e
			}
		} else {
			vars = append(vars, x)
			if e = ns.Push(&argN); e != nil {
				return 0, e
			}
			if e = ds.Push(&argD); e != nil {
				return 0, e
			}
		}
	}
	if err = qm.Denominator(&constant, d); err != nil {
		return 0, err
	}
	for i := range vars {
		den, e := ds.At(uint32(i))
		if e != nil {
			return 0, e
		}
		if e = z.LCM(d, den, d); e != nil {
			return 0, e
		}
	}
	if err = z.Div(d, &constant.Den, &factor); err != nil {
		return 0, err
	}
	if err = z.Mul(&constant.Num, &factor, &sumConstant); err != nil {
		return 0, err
	}
	for i := range vars {
		den, e := ds.At(uint32(i))
		if e != nil {
			return 0, e
		}
		if e = z.Div(d, den, &factor); e != nil {
			return 0, e
		}
		if e = ns.MultiplyAt(uint32(i), &factor); e != nil {
			return 0, e
		}
	}
	if len(vars) == 0 {
		if err = z.Copy(n, &sumConstant); err != nil {
			return 0, err
		}
		return math.MaxUint32, nil
	}
	coefficients := make([]*numeral.Int, len(vars))
	for i := range coefficients {
		coefficients[i], err = ns.At(uint32(i))
		if err != nil {
			return 0, err
		}
	}
	shell, err := c.engine.Shell()
	if err != nil {
		return 0, err
	}
	variable, err = shell.MkSum(&sumConstant, coefficients, vars)
	if err != nil {
		return 0, err
	}
	if err = z.SetInt32(n, 1); err != nil {
		return 0, err
	}
	return variable, nil
}

func (c *HardwareExprConverter) IsVar(t ast.ASTHandle) (bool, error) {
	if err := c.check(); err != nil {
		return false, err
	}
	_, ok := c.vars[t]
	return ok, nil
}

func (c *HardwareExprConverter) Close() error {
	if c == nil || c.self != c {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return nil
	}
	for _, h := range c.order {
		if _, ok := c.vars[h]; !ok {
			continue
		}
		consumed, err := c.ast.DecRefConsumed(h)
		if consumed {
			delete(c.vars, h)
		}
		if err != nil {
			return err
		}
	}
	c.closed = true
	return nil
}
