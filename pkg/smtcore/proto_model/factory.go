package protomodel

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

// ValueFactory is the concrete theory-value interface used while constructing
// an SMT model.
type ValueFactory interface {
	FamilyID() ast.FamilyID
	SomeValue(ast.ASTHandle) (ast.ASTHandle, error)
	SomeValues(ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error)
	FreshValue(ast.ASTHandle) (ast.ASTHandle, error)
	RegisterValue(ast.ASTHandle) error
	Close() error
}

type basicFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
}

func newBasicFactory(m *ast.ASTManager) *basicFactory {
	return &basicFactory{manager: m, family: m.BasicFamilyID()}
}
func (f *basicFactory) FamilyID() ast.FamilyID { return f.family }
func (f *basicFactory) SomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	k, err := f.manager.GetDeclKind(s)
	if err != nil || k != 0 {
		return ast.ASTHandle{}, err
	}
	return f.manager.MkFalse()
}
func (f *basicFactory) SomeValues(s ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	v1, err := f.SomeValue(s)
	if err != nil || v1.IsNull() {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	v2, err := f.manager.MkTrue()
	return v1, v2, err == nil, err
}
func (f *basicFactory) FreshValue(ast.ASTHandle) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, nil
}
func (f *basicFactory) RegisterValue(ast.ASTHandle) error { return nil }
func (f *basicFactory) Close() error                      { return nil }

type factoryValueSet struct {
	values []ast.ASTHandle
	seen   map[ast.ASTHandle]struct{}
	next   big.Int
}

type scalarFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	kind    uint8
	sets    map[ast.ASTHandle]*factoryValueSet
	sorts   []ast.ASTRef
	values  []ast.ASTRef
	finite  map[ast.ASTHandle]bool
}

type ArithmeticFactory struct{ *scalarFactory }
type BitVectorFactory struct{ *scalarFactory }

const (
	scalarUser uint8 = iota
	scalarArithmetic
	scalarBitVector
)

func newScalarFactory(m *ast.ASTManager, family ast.FamilyID, kind uint8) *scalarFactory {
	return &scalarFactory{manager: m, family: family, kind: kind, sets: make(map[ast.ASTHandle]*factoryValueSet), finite: make(map[ast.ASTHandle]bool)}
}
func NewUserSortFactory(m *ast.ASTManager) ValueFactory {
	return newScalarFactory(m, 4, scalarUser)
}
func NewArithmeticFactory(m *ast.ASTManager) (*ArithmeticFactory, error) {
	fid, err := m.ArithmeticFamilyID()
	if err != nil {
		return nil, err
	}
	return &ArithmeticFactory{newScalarFactory(m, fid, scalarArithmetic)}, nil
}
func NewBitVectorFactory(m *ast.ASTManager) (*BitVectorFactory, error) {
	fid, err := m.BitVectorUtilityFamilyID()
	if err != nil {
		return nil, err
	}
	return &BitVectorFactory{newScalarFactory(m, fid, scalarBitVector)}, nil
}
func (f *scalarFactory) FamilyID() ast.FamilyID { return f.family }
func (f *scalarFactory) valueSet(s ast.ASTHandle) (*factoryValueSet, error) {
	if set := f.sets[s]; set != nil {
		return set, nil
	}
	ref, err := f.manager.NewRef(s)
	if err != nil {
		return nil, err
	}
	set := &factoryValueSet{seen: make(map[ast.ASTHandle]struct{})}
	f.sets[s] = set
	f.sorts = append(f.sorts, ref)
	return set, nil
}
func (f *scalarFactory) makeValue(n *big.Int, s ast.ASTHandle) (ast.ASTHandle, error) {
	switch f.kind {
	case scalarUser:
		if !n.IsUint64() || n.Uint64() > uint64(^uint32(0)) {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		return f.manager.MkModelValue(uint32(n.Uint64()), s)
	case scalarArithmetic:
		kind, err := f.manager.GetDeclKind(s)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return f.manager.ArithmeticNumeralRat(new(big.Rat).SetInt(n), kind == 1)
	case scalarBitVector:
		bits, err := f.manager.BitVectorSize(s)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return f.manager.BitVectorNumeralRat(new(big.Rat).SetInt(n), bits)
	default:
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
}
func (f *scalarFactory) insert(n *big.Int, s ast.ASTHandle) (ast.ASTHandle, bool, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	v, err := f.makeValue(n, s)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if _, found := set.seen[v]; found {
		return v, false, nil
	}
	ref, err := f.manager.NewRef(v)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	set.seen[v] = struct{}{}
	set.values = append(set.values, v)
	f.values = append(f.values, ref)
	return v, true, nil
}
func (f *scalarFactory) SomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if f.finite[s] && len(set.values) != 0 {
		return set.values[0], nil
	}
	if len(set.values) != 0 {
		return set.values[0], nil
	}
	v, _, err := f.insert(big.NewInt(0), s)
	return v, err
}
func (f *scalarFactory) SomeValues(s ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	if f.finite[s] {
		if len(set.values) < 2 {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, nil
		}
		return set.values[0], set.values[1], true, nil
	}
	candidate := big.NewInt(0)
	for len(set.values) < 2 {
		if _, _, err = f.insert(candidate, s); err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, err
		}
		candidate.Add(candidate, big.NewInt(1))
	}
	return set.values[0], set.values[1], true, nil
}
func (f *scalarFactory) FreshValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	if f.finite[s] {
		return ast.ASTHandle{}, nil
	}
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	start := new(big.Int).Set(&set.next)
	var limit *big.Int
	if size, e := f.manager.GetSortSize(s); e == nil && size.IsFinite() {
		if n, e := size.Size(); e == nil && n < uint64(^uint32(0)) {
			limit = new(big.Int).Add(start, new(big.Int).SetUint64(n))
		}
	}
	for {
		candidate := new(big.Int).Set(&set.next)
		v, fresh, e := f.insert(candidate, s)
		set.next.Add(&set.next, big.NewInt(1))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if fresh {
			return v, nil
		}
		if limit != nil && set.next.Cmp(limit) > 0 {
			return ast.ASTHandle{}, nil
		}
	}
}
func (f *scalarFactory) RegisterValue(v ast.ASTHandle) error {
	s, err := f.manager.GetSort(v)
	if err != nil {
		return err
	}
	return f.registerForSort(v, s)
}
func (f *scalarFactory) registerForSort(v, s ast.ASTHandle) error {
	set, err := f.valueSet(s)
	if err != nil {
		return err
	}
	if _, found := set.seen[v]; found {
		return nil
	}
	ref, err := f.manager.NewRef(v)
	if err != nil {
		return err
	}
	set.seen[v] = struct{}{}
	set.values = append(set.values, v)
	f.values = append(f.values, ref)
	return nil
}
func (f *ArithmeticFactory) MkNumValue(value *big.Rat, integer bool) (ast.ASTHandle, error) {
	v, err := f.manager.ArithmeticNumeralRat(value, integer)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	s, err := f.manager.GetSort(v)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return v, f.registerForSort(v, s)
}
func (f *BitVectorFactory) MkNumValue(value *big.Rat, bits uint32) (ast.ASTHandle, error) {
	v, err := f.manager.BitVectorNumeralRat(value, bits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	s, err := f.manager.GetSort(v)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return v, f.registerForSort(v, s)
}
func (f *scalarFactory) Freeze(s ast.ASTHandle) error {
	set, err := f.valueSet(s)
	if err != nil {
		return err
	}
	if len(set.values) == 0 {
		if _, err = f.SomeValue(s); err != nil {
			return err
		}
	}
	f.finite[s] = true
	return nil
}
func (f *scalarFactory) IsFinite(s ast.ASTHandle) bool { return f.finite[s] }
func (f *scalarFactory) KnownUniverse(s ast.ASTHandle) []ast.ASTHandle {
	if set := f.sets[s]; set != nil {
		return append([]ast.ASTHandle(nil), set.values...)
	}
	return nil
}
func (f *scalarFactory) Sorts() []ast.ASTHandle {
	result := make([]ast.ASTHandle, 0, len(f.sorts))
	for _, ref := range f.sorts {
		h, _ := f.manager.RefGet(ref)
		result = append(result, h)
	}
	return result
}
func (f *scalarFactory) Close() (err error) {
	for i := len(f.values) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.values[i]))
	}
	for i := len(f.sorts) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.sorts[i]))
	}
	f.values, f.sorts, f.sets, f.finite = nil, nil, nil, nil
	return err
}
