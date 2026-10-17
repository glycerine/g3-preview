// Source: polynomial_var2value.h:28-44, concrete instantiation retained by
// cmd_context/extra_cmds/polynomial_cmds.cpp:93.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"reflect"
)

// SimpleAlgebraicValues owns copies in insertion order. Duplicate variables
// remain separate entries and lookup selects the first one, as in the source.
// Returned cells are borrowed until the next mutation or Close.
type SimpleAlgebraicValues struct {
	self                     *SimpleAlgebraicValues
	core                     *AlgebraicRootCore
	variables                []uint32
	values                   *AlgebraicVector
	header                   uint64
	running, closing, closed bool
}

var simpleAlgebraicValuesZero Algebraic
var _ PolynomialAlgebraicEvaluation = (*SimpleAlgebraicValues)(nil)

func NewSimpleAlgebraicValues(core *AlgebraicRootCore) (*SimpleAlgebraicValues, error) {
	if e := core.check(); e != nil {
		return nil, e
	}
	b := core.config.Rationals.Budget()
	h := uint64(reflect.TypeOf(SimpleAlgebraicValues{}).Size())
	if e := b.Reserve(h); e != nil {
		return nil, e
	}
	values, e := core.NewVector()
	if e != nil {
		b.Release(h)
		return nil, e
	}
	s := &SimpleAlgebraicValues{core: core, values: values, header: h}
	s.self = s
	return s, nil
}
func (s *SimpleAlgebraicValues) check() error {
	if s == nil || s.self != s {
		return ErrAlgebraicOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	if s.running {
		return ErrAlgebraicPending
	}
	return s.values.check()
}
func (s *SimpleAlgebraicValues) Manager() (*AlgebraicRootCore, error) {
	if e := s.check(); e != nil {
		return nil, e
	}
	return s.core, nil
}
func (s *SimpleAlgebraicValues) Push(x uint32, v *Algebraic) error {
	if e := s.check(); e != nil {
		return e
	}
	s.running = true
	defer func() { s.running = false }()
	// Preserve the published variable prefix when the subsequent value copy fails.
	s.variables = append(s.variables, x)
	return s.values.PushCopy(v)
}
func (s *SimpleAlgebraicValues) Contains(x uint32) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	for _, v := range s.variables {
		if v == x {
			return true, nil
		}
	}
	return false, nil
}
func (s *SimpleAlgebraicValues) Algebraic(x uint32) (*Algebraic, error) {
	if e := s.check(); e != nil {
		return nil, e
	}
	for i, v := range s.variables {
		if v == x {
			if i >= len(s.values.values) {
				return nil, arena.ErrBounds
			}
			return &s.values.values[i], nil
		}
	}
	if e := diagnostic.BorrowProcessDebugContext().UnreachableSource("src/math/polynomial/polynomial_var2value.h", 41); e != nil {
		return nil, e
	}
	return &simpleAlgebraicValuesZero, nil
}
func (s *SimpleAlgebraicValues) Close() error {
	if s == nil || s.self != s {
		return ErrAlgebraicOwner
	}
	if s.closed {
		return nil
	}
	if s.running {
		return ErrAlgebraicPending
	}
	s.closing = true
	if e := s.values.Close(); e != nil {
		return e
	}
	s.variables = nil
	s.core.config.Rationals.Budget().Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
