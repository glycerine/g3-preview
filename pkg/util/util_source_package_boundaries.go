// Source value and borrowed-owner boundaries used by AST and parameter consumers.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"reflect"
)

// SymbolWords encodes the pointer-free identity in source aggregate storage.
// DecodeSymbolWords validates the reconstructed identity against this manager.
func (s Symbol) SymbolWords() [5]uint32 {
	return [5]uint32{uint32(s.owner), uint32(s.owner >> 32), s.Value, uint32(s.Generation), s.Tag}
}
func (m *SymbolManager) DecodeSymbolWords(v [5]uint32) (Symbol, error) {
	return m.FromToken(Symbol{owner: uint64(v[0]) | uint64(v[1])<<32, Value: v[2], Generation: arena.Generation(v[3]), Tag: v[4]}.Token())
}
func (s Symbol) IsNull() bool { return s.Tag == 0 && s.Value == 0 }

// CStringView borrows the source bare_str storage; numerical symbols follow
// the source debug guard and release empty spelling. No copy or RC is acquired.
func (m *SymbolManager) CStringView(s Symbol) (ParamsStringView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return ParamsStringView{}, e
	}
	if s.IsNumerical() {
		if sourceDebug {
			return ParamsStringView{}, ErrInvalidSymbol
		}
		return LiteralParamsStringView(""), nil
	}
	if s.Value == 0 {
		return ParamsStringView{}, ErrInvalidSymbol
	}
	return ParamsStringView{region: m.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(s.Value), Generation: s.Generation}, offset: m.pointerWidth, Mode: 3, PresentField: true}, nil
}
func (c *ParamsContext) CheckServices(s *SymbolManager, q *RationalService) error {
	if c == nil || c.Self != c || c.SymbolsField != s || q != nil && c.NumeralsField != q {
		return ErrParamsOwner
	}
	return nil
}
func (c *ParamsContext) Symbols() *SymbolManager    { return c.SymbolsField }
func (c *ParamsContext) Numerals() *RationalService { return c.NumeralsField }
func (c *ParamsContext) Budget() *arena.Budget      { return c.BudgetField }
func (g *GlobalParamsService) CheckOwner(c *ParamsContext, b *arena.Budget) error {
	if g == nil || g.self != g || g.Context != c || g.Config.Budget != b {
		return ErrParamsOwner
	}
	return nil
}
func (r *Reslimit) InitEmbedded(b *arena.Budget) { r.budget = b }
func (r *Reslimit) StorageBytes() uint64 {
	return uint64(cap(r.Limits))*8 + uint64(cap(r.Children))*uint64(reflect.TypeOf((*Reslimit)(nil)).Size())
}
func (g *GlobalParamsService) CheckBudget(b *arena.Budget) error {
	if g == nil || g.self != g || g.Config.Budget != b {
		return ErrGlobalParamsOwner
	}
	return nil
}
func (g *GlobalParamsService) NamespaceVerifyFailure(file string, line int32, expression string) error {
	if g.namespaceAssertions == nil {
		return ErrGlobalParamsAssertion
	}
	return g.namespaceAssertions.VerifyFailure(file, line, expression)
}

// NamespaceAssertionFailureOptional preserves the enabled-default nil binding.
func NamespaceAssertionFailureOptional(g *GlobalParamsService, file string, line int32, expression string) error {
	if g == nil {
		return ErrGlobalParamsAssertion
	}
	return g.NamespaceAssertionFailure(file, line, expression)
}
func (c *ParamsContext) CheckOpen() error         { return c.open() }
func (c *ParamsContext) Check(p *ParamsRef) error { return c.check(p) }
func (g *GlobalParamsService) CheckOpen() error   { return g.open() }

// EmptyValue constructs a null-backed params_ref without allocating a block.
func (c *ParamsContext) EmptyValue() ParamsRef { return ParamsRef{Owner: c.Owner} }

func (r ParamsRef) Initialized() bool { return r.Owner != 0 }

func (g *GlobalParams) CheckOpen() error        { return g.open() }
func (g *GlobalParams) Context() *ParamsContext { return g.ContextField }
func (g *GlobalParams) Symbols() *SymbolManager { return g.SymbolsField }
func (r *Reslimit) Closed() bool                { return r.ClosedField }
