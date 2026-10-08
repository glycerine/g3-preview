// Source: src/ast/rewriter/bv_rewriter.cpp:1362-1406;
// src/util/ref_vector.h:41-55,96-101,128-132.
package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	"reflect"
)

// Original expr_ref_vector: INC precedes push/growth and replacement DEC.
// Retain an acquired replacement if DEC stops, recording consumption before
// returning so owning cleanup cannot decrement the old reference twice.
type bv2IntRefVector struct {
	values  ast.ASTHandleVector
	pending ast.ASTHandle
}

func (v *bv2IntRefVector) push(m *ast.ASTManager, h ast.ASTHandle) error {
	if !v.pending.IsNull() {
		return ast.ErrASTPending
	}
	if e := m.IncRef(h); e != nil {
		return e
	}
	v.pending = h
	if e := v.values.Push(m.Budget(), h); e != nil {
		return e
	}
	v.pending = ast.ASTHandle{}
	return nil
}
func (v *bv2IntRefVector) set(m *ast.ASTManager, i uint32, h ast.ASTHandle) error {
	if !v.pending.IsNull() || i >= uint32(len(v.values.Slice())) {
		return ast.ErrASTPending
	}
	if e := m.IncRef(h); e != nil {
		return e
	}
	v.pending = h
	consumed, e := m.DecRefConsumed(v.values.Slice()[i])
	if consumed {
		v.values.Slice()[i] = ast.ASTHandle{}
	}
	if e != nil {
		return e
	}
	v.values.Slice()[i] = h
	v.pending = ast.ASTHandle{}
	return nil
}
func (v *bv2IntRefVector) endPending(m *ast.ASTManager) error {
	if !v.pending.IsNull() {
		consumed, e := m.DecRefConsumed(v.pending)
		if consumed {
			v.pending = ast.ASTHandle{}
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (v *bv2IntRefVector) close(m *ast.ASTManager) error {
	if e := v.endPending(m); e != nil {
		return e
	}
	for i, h := range v.values.Slice() {
		if h.IsNull() {
			continue
		}
		consumed, e := m.DecRefConsumed(h)
		if consumed {
			v.values.Slice()[i] = ast.ASTHandle{}
		}
		if e != nil {
			return e
		}
	}
	v.values.Close(m.Budget())
	return nil
}

type bv2IntInvocation struct {
	numerals *ast.BitVectorBV2IntNumerals
	args     bv2IntRefVector
	tmp      ast.ASTRef
	pair     [2]ast.ASTHandle
	header   uint64
}

func (v *bv2IntInvocation) close(m *ast.ASTManager) error {
	if v.numerals != nil {
		if e := v.numerals.CloseExpression(); e != nil {
			return e
		}
	}
	if e := v.args.endPending(m); e != nil {
		return e
	}
	if v.tmp != (ast.ASTRef{}) {
		if e := m.DelRef(&v.tmp); e != nil {
			return e
		}
	}
	if e := v.args.close(m); e != nil {
		return e
	}
	if v.numerals != nil {
		if e := v.numerals.Close(); e != nil {
			return e
		}
		v.numerals = nil
	}
	m.Budget().Release(v.header)
	v.header = 0
	return nil
}

func (p *BVRewriter) RewriteBV2Int(arg ast.ASTHandle, out ast.ASTRef) (st RewriteStatus, err error) {
	if e := p.open(); e != nil {
		return RewriteFailed, e
	}
	p.running = true
	st = RewriteFailed
	defer func() { p.running = false; p.fail(err) }()
	m := p.manager
	bytes := uint64(reflect.TypeOf(bv2IntInvocation{}).Size())
	if e := m.Budget().Reserve(bytes); e != nil {
		return st, e
	}
	v := &bv2IntInvocation{header: bytes}
	p.bv2int = v
	n, e := ast.NewBitVectorBV2IntNumerals(m)
	v.numerals = n
	if e != nil {
		return st, e
	}
	var width uint32
	numeral, e := n.Read(arg, &width)
	if e != nil {
		return st, e
	}
	var result ast.ASTHandle
	if numeral {
		result, e = n.Constant(0)
		if e != nil {
			return st, e
		}
		st = RewriteDone
	} else {
		concat, x := m.IsBitVectorOperation(arg, 37)
		if x != nil {
			return st, x
		}
		if concat {
			count, x := m.GetNumArgs(arg)
			if x != nil {
				return st, x
			}
			if count == 0 {
				result, e = n.Zero()
				if e != nil {
					return st, e
				}
				st = RewriteDone
			} else {
				if e = p.bv2IntChildren(v, arg, count); e != nil {
					return st, e
				}
				last, x := m.GetArg(arg, count-1)
				if x != nil {
					return st, x
				}
				size, x := m.BitVectorSize(last)
				if x != nil {
					return st, x
				}
				for i := count - 1; i > 0; {
					v.tmp, e = m.NewRef(ast.ASTHandle{})
					if e != nil {
						return st, e
					}
					i--
					if e = m.SetRef(v.tmp, v.args.values.Slice()[i]); e != nil {
						return st, e
					}
					v.pair[0], e = n.Coefficient(size)
					if e != nil {
						return st, e
					}
					v.pair[1], e = m.RefGet(v.tmp)
					if e != nil {
						return st, e
					}
					product, x := m.MkFamilyApp(p.arithFamily, 9, nil, v.pair[:])
					if x != nil {
						return st, x
					}
					if e = m.SetRef(v.tmp, product); e != nil {
						return st, e
					}
					// Full-expression power and literal two end after tmp assignment.
					if e = n.End(2, 1); e != nil {
						return st, e
					}
					h, x := m.RefGet(v.tmp)
					if x != nil {
						return st, x
					}
					if e = v.args.set(m, i, h); e != nil {
						return st, e
					}
					child, x := m.GetArg(arg, i)
					if x != nil {
						return st, x
					}
					w, x := m.BitVectorSize(child)
					if x != nil {
						return st, x
					}
					size += w
					if e = m.DelRef(&v.tmp); e != nil {
						return st, e
					}
				}
				result, e = p.bv2IntArithmeticResult(v, 6)
				if e != nil {
					return st, e
				}
				st = Rewrite2
			}
		} else {
			mul, x := p.isMultiplyWithoutOverflow(arg)
			if x != nil {
				return st, x
			}
			add := false
			if !mul {
				add, x = p.isAddWithoutOverflow(arg)
				if x != nil {
					return st, x
				}
			}
			if mul || add {
				count, x := m.GetNumArgs(arg)
				if x != nil {
					return st, x
				}
				if e = p.bv2IntChildren(v, arg, count); e != nil {
					return st, e
				}
				kind := int32(6)
				if mul {
					kind = 9
				}
				result, e = p.bv2IntArithmeticResult(v, kind)
				if e != nil {
					return st, e
				}
				st = Rewrite2
			}
		}
	}
	if e != nil {
		return st, e
	}
	if st != RewriteFailed {
		if e = m.SetRef(out, result); e != nil {
			return st, e
		}
	}
	// Source vector is younger than default v.
	if e = v.args.close(m); e != nil {
		return st, e
	}
	if e = n.SourceEnd(); e != nil {
		return st, e
	}
	if e = v.close(m); e != nil {
		return st, e
	}
	p.bv2int = nil
	return st, nil
}
func (p *BVRewriter) bv2IntChildren(v *bv2IntInvocation, arg ast.ASTHandle, count uint32) error {
	for i := uint32(0); i < count; i++ {
		x, e := p.manager.GetArg(arg, i)
		if e != nil {
			return e
		}
		h, e := p.manager.MkBitVectorToInt(x)
		if e != nil {
			return e
		}
		if e = v.args.push(p.manager, h); e != nil {
			return e
		}
	}
	return nil
}

// Source: src/ast/arith_decl_plugin.h:373,381.
func (p *BVRewriter) bv2IntArithmeticResult(v *bv2IntInvocation, kind int32) (ast.ASTHandle, error) {
	hs := v.args.values.Slice()
	// Every child was constructed as a bv2int app (or a coefficient MUL app).
	if len(hs) == 1 {
		return hs[0], nil
	}
	return p.manager.MkFamilyApp(p.arithFamily, kind, nil, hs)
}
