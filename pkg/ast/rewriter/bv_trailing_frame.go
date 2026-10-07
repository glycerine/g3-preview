// Source: src/ast/rewriter/bv_trailing.cpp:70-82,121-129,142-171,183-208.
package rewriter

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

// Each call names its reached structural frame and numerical child. A parent
// loop reuses this call site only after a successful, fully retired return.
type bvTrailingCall struct {
	child   *bvTrailingFrame
	numeral ast.ASTBitVectorNumeralInvocation
}

func (c *bvTrailingCall) sourceReady(m *ast.ASTManager) error {
	if e := m.BitVectorNumeralInvocationSourceReady(&c.numeral); e != nil {
		return e
	}
	if f := c.child; f != nil {
		if f.sourceCause != nil {
			return f.sourceCause
		}
		if !f.sourceDone {
			return ast.ErrASTPending
		}
	}
	return nil
}
func (c *bvTrailingCall) cleanup(m *ast.ASTManager) error {
	if f := c.child; f != nil && !f.retired {
		if e := f.destroy(true); e != nil {
			return e
		}
	}
	return m.CleanupBitVectorNumeralInvocation(&c.numeral)
}

const (
	bvTrailingAddFrame = iota
	bvTrailingMulFrame
	bvTrailingConcatFrame
	bvTrailingEqFrame
)

// Checked Go wrapper storage is charged; native inline expr_ref/vector stack
// layout is not claimed. Ref/vector DEC consumption remains in AST's owners.
type bvTrailingFrame struct {
	owner                   *BVTrailing
	next                    *bvTrailingFrame
	header                  uint64
	mode                    int
	refs                    [2]ast.ASTRef
	vector, pushVector      ast.ASTRefVector
	vectorPlaced            bool
	call                    bvTrailingCall
	scope                   ast.ASTBitVectorNumeralScope
	scopePlaced, scopeEnded bool
	push                    ast.ASTRefVectorPushReceipt
	pushArg                 ast.ASTHandle
	args                    ast.ASTHandleBuffer
	sourceDone, retired     bool
	bodyCause, sourceCause  error
}

func (p *BVTrailing) beginFrame(mode int, call *bvTrailingCall) (*bvTrailingFrame, error) {
	n := uint64(reflect.TypeOf(bvTrailingFrame{}).Size())
	if e := p.manager.Budget().Reserve(n); e != nil {
		return nil, e
	}
	f := &bvTrailingFrame{owner: p, next: p.frames, header: n, mode: mode}
	p.frames = f
	call.child = f
	return f, nil
}
func (f *bvTrailingFrame) newRef(i int) error {
	r, e := f.owner.manager.NewRef(ast.ASTHandle{})
	if e == nil {
		f.refs[i] = r
	}
	return e
}
func (f *bvTrailingFrame) newVector() error {
	v, e := f.owner.manager.NewRefVector()
	if e == nil {
		f.vector, f.pushVector, f.vectorPlaced = v, v, true
	}
	return e
}
func (f *bvTrailingFrame) pushArgValue(h ast.ASTHandle) error {
	f.pushArg = h
	if e := f.owner.manager.RefVectorPushTracked(f.vector, h, &f.push); e != nil {
		return e
	}
	// Publication transferred the exact INC to vector storage. Failed push
	// receipts remain address-stable for separate owning acknowledgment.
	f.push = ast.ASTRefVectorPushReceipt{}
	return nil
}
func (f *bvTrailingFrame) makeApp(original ast.ASTHandle, kind int32) (ast.ASTHandle, error) {
	m := f.owner.manager
	n, e := m.RefVectorSize(f.vector)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	for i := uint32(0); i < n; i++ {
		h, e := m.RefVectorGet(f.vector, i)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e = f.args.Push(m.Budget(), h); e != nil {
			return ast.ASTHandle{}, e
		}
	}
	fid, e := m.GetFamilyID(original)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	// This checked interop buffer bridges compact vector storage and the
	// existing slice factory. It adds no AST refs; its backing is charged.
	return m.MkFamilyApp(fid, kind, nil, f.args.Slice())
}
func (f *bvTrailingFrame) delRef(i int) error {
	if f.refs[i] == (ast.ASTRef{}) {
		return nil
	}
	return f.owner.manager.DelRef(&f.refs[i])
}
func (f *bvTrailingFrame) delVector() error {
	if !f.vectorPlaced {
		return nil
	}
	if e := f.owner.manager.DelRefVector(&f.vector); e != nil {
		return e
	}
	f.vectorPlaced = false
	return nil
}
func (f *bvTrailingFrame) destroy(own bool) error {
	if f.retired || !own && f.sourceDone {
		return nil
	}
	if !own && f.sourceCause != nil {
		return f.sourceCause
	}
	m := f.owner.manager
	if own {
		if e := f.call.cleanup(m); e != nil {
			return e
		}
		held, e := f.push.Unpublished()
		if e != nil {
			return e
		}
		if held {
			if e := m.CompleteRefVectorPush(f.pushVector, f.pushArg, &f.push); e != nil {
				return e
			}
		}
		if f.scopePlaced {
			if e := m.CleanupBitVectorNumeralScope(&f.scope); e != nil {
				return e
			}
		}
	} else {
		if e := f.call.sourceReady(m); e != nil {
			return e
		}
		if f.scopePlaced {
			if e := m.BitVectorNumeralScopeSourceReady(&f.scope); e != nil {
				return e
			}
		}
	}
	f.args.Close(m.Budget())
	switch f.mode {
	case bvTrailingAddFrame:
		if e := f.delRef(0); e != nil {
			return e
		}
		if e := f.delVector(); e != nil {
			return e
		}
	case bvTrailingMulFrame, bvTrailingConcatFrame:
		if e := f.delVector(); e != nil {
			return e
		}
		if e := f.delRef(0); e != nil {
			return e
		}
	case bvTrailingEqFrame:
		if e := f.delRef(1); e != nil {
			return e
		}
		if e := f.delRef(0); e != nil {
			return e
		}
	default:
		return ast.ErrASTDomain
	}
	if !own {
		f.sourceDone = true
	}
	return nil
}
func (f *bvTrailingFrame) finish(body *error) {
	f.bodyCause = *body
	// A stopped exact younger child blocks the source footer, including
	// the multiplication caller's older c_val/vector/tmp sequence.
	e := f.call.sourceReady(f.owner.manager)
	if e == nil && f.scopePlaced && !f.scopeEnded {
		f.scopeEnded = true
		*body = f.owner.manager.EndBitVectorNumeralScope(&f.scope, *body)
	}
	if e == nil {
		e = f.destroy(false)
	}
	f.sourceCause = e
	*body = errors.Join(*body, e)
	if *body == nil && f.sourceDone {
		*body = f.destroy(true)
		if *body == nil {
			p := f.owner
			if p.frames != f {
				*body = ast.ErrASTPending
				return
			}
			p.frames = f.next
			p.manager.Budget().Release(f.header)
			f.header = 0
			f.retired = true
		}
	}
}
