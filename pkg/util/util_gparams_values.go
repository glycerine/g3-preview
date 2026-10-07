package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Actual mutable normalization scratch; retained capacity is an explicit cold
// Go workspace adapter. Every backing/overlap is charged, no hot strings kept.

func (g *GlobalParams) reserveWorkspace(n int) error {
	if n < 0 {
		return arena.ErrBounds
	}
	if n <= cap(g.Workspace) {
		return nil
	}
	next := cap(g.Workspace)
	if next == 0 {
		next = 2
	}
	for next < n {
		if next > int(^uint(0)>>1)/2 {
			next = n
			break
		}
		next *= 2
	}
	if e := g.Budget.Reserve(uint64(next)); e != nil {
		return e
	}
	b := make([]byte, len(g.Workspace), next)
	copy(b, g.Workspace)
	g.Budget.Release(uint64(cap(g.Workspace)))
	g.Workspace = b
	return nil
}

// Go conversion can allocate a transient immutable copy in addition to actual
// mutable scratch; reserve its exact byte extent before conversion and refund
// it after interning. No hidden name string survives this temporary scope.
func (g *GlobalParams) internWorkspace(b []byte) (Symbol, error) {
	if len(b) == 0 {
		return g.SymbolsField.Intern("")
	}
	if e := g.Budget.Reserve(uint64(len(b))); e != nil {
		return Symbol{}, e
	}
	text := string(b)
	k, e := g.SymbolsField.Intern(text)
	g.Budget.Release(uint64(len(b)))
	return k, e
}
func (g *GlobalParams) normalizeWorkspace() (Symbol, Symbol, error) {
	n := uint32(len(g.Workspace)) // source unsigned narrowing, before both loops
	for i := uint32(0); i < n; i++ {
		ch := g.Workspace[i]
		if ch >= 'A' && ch <= 'Z' {
			g.Workspace[i] = ch - 'A' + 'a'
		} else if ch == '-' {
			g.Workspace[i] = '_'
		}
	}
	for i := uint32(0); i < n; i++ {
		if g.Workspace[i] == '.' {
			// The source publishes suffix PARAM before prefix MODULE.
			param, e := g.internWorkspace(g.Workspace[i+1:])
			if e != nil {
				return Symbol{}, Symbol{}, e
			}
			module, e := g.internWorkspace(g.Workspace[:i])
			return module, param, e
		}
	}
	param, e := g.internWorkspace(g.Workspace)
	return Symbol{}, param, e
}
func (g *GlobalParams) normalizeName(name string) (Symbol, Symbol, error) {
	name = name[:cSymbolLength(name)]
	if len(name) > 0 && name[0] == ':' {
		name = name[1:]
	}
	if e := g.reserveWorkspace(len(name)); e != nil {
		return Symbol{}, Symbol{}, e
	}
	g.Workspace = g.Workspace[:len(name)]
	copy(g.Workspace, name)
	return g.normalizeWorkspace()
}
func (g *GlobalParams) normalizeSymbol(name Symbol) (Symbol, Symbol, error) {
	m := g.SymbolsField
	m.mu.Lock()
	if e := m.checkPlain(name); e != nil {
		m.mu.Unlock()
		return Symbol{}, Symbol{}, e
	}
	if name.IsNumerical() {
		m.mu.Unlock()
		if GlobalParamsSourceDebug {
			return Symbol{}, Symbol{}, ErrGlobalParamsOwner
		}
		return g.normalizeName("")
	}
	if name.Value == 0 {
		m.mu.Unlock()
		return Symbol{}, Symbol{}, ErrGlobalParamsOwner
	}
	g.Workspace = g.Workspace[:0]
	r := arena.BytesRegionRef{Slot: arena.Slot(name.Value), Generation: name.Generation}
	first := true
	for i := m.pointerWidth; ; i++ {
		b, e := m.region.Read(r, i)
		if e != nil {
			m.mu.Unlock()
			return Symbol{}, Symbol{}, e
		}
		if b == 0 {
			break
		}
		if first && b == ':' {
			first = false
			continue
		}
		first = false
		if e = g.reserveWorkspace(len(g.Workspace) + 1); e != nil {
			m.mu.Unlock()
			return Symbol{}, Symbol{}, e
		}
		g.Workspace = append(g.Workspace, b)
	}
	m.mu.Unlock()
	return g.normalizeWorkspace()
}

type gparamsErrorPart struct {
	text     string
	symbol   Symbol
	symbolic bool
}

func gpText(s string) gparamsErrorPart   { return gparamsErrorPart{text: s} }
func gpSymbol(s Symbol) gparamsErrorPart { return gparamsErrorPart{symbol: s, symbolic: true} }

// Parts are transient cold stack arguments, never records. Returned immutable
// exception/object/text are caller-owned; actual composition owner is charged.
func (g *GlobalParams) exception(parts ...gparamsErrorPart) error {
	b, e := NewStringBuffer64(StringBufferConfig{Budget: g.Budget})
	if e != nil {
		return e
	}
	defer b.Close()
	for _, p := range parts {
		if p.symbolic {
			e = b.AppendSymbol(g.SymbolsField, p.symbol)
		} else {
			e = b.AppendCString(p.text)
		}
		if e != nil {
			return e
		}
	}
	text, e := b.CStr()
	if e != nil {
		return e
	}
	return NewDefaultException(text)
}
func (g *GlobalParams) oldName(k Symbol) (bool, error) {
	for _, name := range gparamsOldNames {
		same, e := g.SymbolsField.EqualString(k, &name)
		if e != nil {
			return false, e
		}
		if same {
			return true, nil
		}
	}
	return false, nil
}
func (g *GlobalParams) renamedName(k Symbol) (string, bool, error) {
	for _, pair := range gparamsRenames {
		same, e := g.SymbolsField.EqualString(k, &pair.old)
		if e != nil {
			return "", false, e
		}
		if same {
			return pair.new, true, nil
		}
	}
	return "", false, nil
}
func (g *GlobalParams) unknownParameter(k Symbol, d *ParamDescrs, module Symbol) error {
	b, e := NewStringBuffer64(StringBufferConfig{Budget: g.Budget})
	if e != nil {
		return e
	}
	defer b.Close()
	if module == (Symbol{}) {
		newName, found, e := g.renamedName(k)
		if e != nil {
			return e
		}
		if found {
			if e = b.AppendCString("the parameter '"); e != nil {
				return e
			}
			if e = b.AppendSymbol(g.SymbolsField, k); e != nil {
				return e
			}
			if e = b.AppendCString("', invoke 'z3 -p' to obtain the new parameter list, and 'z3 -pp:"); e != nil {
				return e
			}
			if e = b.AppendCString(newName); e != nil {
				return e
			}
			if e = b.AppendCString("' for the full description of the parameter"); e != nil {
				return e
			}
			text, e := b.CStr()
			if e != nil {
				return e
			}
			return NewDefaultException(text)
		}
		old, e := g.oldName(k)
		if e != nil {
			return e
		}
		if old {
			return g.exception(gpText("unknown parameter '"), gpSymbol(k), gpText("', this is an old parameter name, invoke 'z3 -p' to obtain the new parameter list"))
		}
	}
	if e = b.AppendCString("unknown parameter '"); e != nil {
		return e
	}
	if e = b.AppendSymbol(g.SymbolsField, k); e != nil {
		return e
	}
	if module == (Symbol{}) {
		e = b.AppendCString("'\n")
	} else {
		if e = b.AppendCString("' at module '"); e != nil {
			return e
		}
		if e = b.AppendSymbol(g.SymbolsField, module); e != nil {
			return e
		}
		e = b.AppendCString("'\n")
	}
	if e != nil {
		return e
	}
	if e = b.AppendCString("Legal parameters are:\n"); e != nil {
		return e
	}
	if e = d.DisplayInto(b, 2, false, false); e != nil {
		return e
	}
	text, e := b.CStr()
	if e != nil {
		return e
	}
	return NewDefaultException(text)
}
func (g *GlobalParams) findModuleDescriptor(k Symbol) (*ParamDescrs, bool, error) {
	if e := g.ensureRegistered(); e != nil {
		return nil, false, e
	}
	h, e := g.SymbolsField.Hash(k)
	if e != nil {
		return nil, false, e
	}
	i := g.ModuleDescriptors.find(k, h)
	if i < 0 {
		return nil, false, nil
	}
	n := g.ModuleDescriptors.Entries[i].descriptor
	if n == 0 || int(n) > len(g.Descriptors) || g.Descriptors[n-1] == nil {
		return nil, false, ErrGlobalParamsOwner
	}
	return g.Descriptors[n-1], true, nil
}
func (g *GlobalParams) validateType(k Symbol, value string, d *ParamDescrs) error {
	kind, e := d.Kind(k)
	if e != nil {
		return e
	}
	switch kind {
	case ParamUInt:
		for i := 0; i < len(value); i++ {
			if value[i] < '0' || value[i] > '9' {
				return g.exception(gpText("Expected values for parameter "), gpSymbol(k), gpText(" is an unsigned integer. It was given argument '"), gpText(value), gpText("'"))
			}
		}
	case ParamDouble:
		for i := 0; i < len(value); i++ {
			b := value[i]
			if (b < '0' || b > '9') && b != '.' && b != '-' && b != '/' {
				return g.exception(gpText("Expected values for parameter "), gpSymbol(k), gpText(" is a double. It was given argument '"), gpText(value), gpText("'"))
			}
		}
	case ParamBool:
		if value != "true" && value != "false" {
			return g.exception(gpText("Expected values for parameter "), gpSymbol(k), gpText(" are 'true' or 'false'. It was given argument '"), gpText(value), gpText("'"))
		}
	}
	return nil
}
func (g *GlobalParams) setValue(k Symbol, value string, d *ParamDescrs, module Symbol) error {
	kind, e := d.Kind(k)
	if e != nil {
		return e
	}
	// Source publication intentionally precedes the invalid-kind diagnostic.
	ps, e := g.GetParams(module)
	if e != nil {
		return e
	}
	switch kind {
	case ParamInvalid:
		return g.unknownParameter(k, d, module)
	case ParamUInt:
		return g.ContextField.SetUInt(ps, k, gparamsParseUint(value, g.cLongBits))
	case ParamDouble:
		return g.ContextField.SetDouble(ps, k, gparamsParseDouble(value))
	case ParamBool:
		if value == "true" {
			return g.ContextField.SetBool(ps, k, true)
		}
		if value == "false" {
			return g.ContextField.SetBool(ps, k, false)
		}
		if module == (Symbol{}) {
			return g.exception(gpText("invalid value '"), gpText(value), gpText("' for Boolean parameter '"), gpSymbol(k), gpText("' at module '"), gpSymbol(module), gpText("'"))
		}
		return g.exception(gpText("invalid value '"), gpText(value), gpText("' for Boolean parameter '"), gpSymbol(k), gpText("'"))
	case ParamSymbol:
		s, e := g.SymbolsField.Intern(value)
		if e != nil {
			return e
		}
		return g.ContextField.SetSymbol(ps, k, s)
	case ParamString:
		s, e := g.SymbolsField.Intern(value)
		if e != nil {
			return e
		}
		return g.ContextField.SetStringSymbol(ps, k, s)
	default:
		if module == (Symbol{}) {
			return g.exception(gpText("unsupported parameter type '"), gpSymbol(k), gpText("' at module '"), gpSymbol(module), gpText("'"))
		}
		return g.exception(gpText("unsupported parameter type '"), gpSymbol(k), gpText("'"))
	}
}
func (g *GlobalParams) setNormalized(module, k Symbol, value string) error {
	var d *ParamDescrs
	if module == (Symbol{}) {
		if e := g.ensureRegistered(); e != nil {
			return e
		}
		d = g.GlobalDescriptors
	} else {
		found, ok, e := g.findModuleDescriptor(module)
		if e != nil {
			return e
		}
		if !ok {
			return g.exception(gpText("invalid parameter, unknown module '"), gpSymbol(module), gpText("'"))
		}
		d = found
	}
	value = value[:cSymbolLength(value)]
	if e := g.validateType(k, value, d); e != nil {
		return e
	}
	return g.setValue(k, value, d, module)
}
func (g *GlobalParams) SetName(name, value string) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	module, k, e := g.normalizeName(name)
	if e != nil {
		return e
	}
	return g.setNormalized(module, k, value)
}
func (g *GlobalParams) SetSymbol(name Symbol, value string) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	module, k, e := g.normalizeSymbol(name)
	if e != nil {
		return e
	}
	return g.setNormalized(module, k, value)
}
func (g *GlobalParams) getDefault(d *ParamDescrs, k, module Symbol) (string, error) {
	yes, e := d.Contains(k)
	if e != nil {
		return "", e
	}
	if !yes {
		return "", g.unknownParameter(k, d, module)
	}
	s, present, e := d.Default(k)
	if e != nil {
		return "", e
	}
	if !present {
		return "default", nil
	}
	return s, nil
}
func (g *GlobalParams) getValue(module, k Symbol) (string, error) {
	if module == (Symbol{}) {
		yes, e := g.ContextField.Contains(&g.globalParams, k)
		if e != nil {
			return "", e
		}
		if yes {
			return g.ContextField.DisplayKey(&g.globalParams, k)
		}
		if e = g.ensureRegistered(); e != nil {
			return "", e
		}
		return g.getDefault(g.GlobalDescriptors, k, module)
	}
	h, e := g.SymbolsField.Hash(module)
	if e != nil {
		return "", e
	}
	if i := g.ModuleParams.find(module, h); i >= 0 {
		n := g.ModuleParams.Entries[i].cell
		if n == 0 || int(n) > len(g.Cells) || g.Cells[n-1] == nil {
			return "", ErrGlobalParamsOwner
		}
		ps := &g.Cells[n-1].value
		yes, e := g.ContextField.Contains(ps, k)
		if e != nil {
			return "", e
		}
		if yes {
			return g.ContextField.DisplayKey(ps, k)
		}
	}
	d, yes, e := g.findModuleDescriptor(module)
	if e != nil {
		return "", e
	}
	if !yes {
		return "", g.exception(gpText("unknown module '"), gpSymbol(module), gpText("'"))
	}
	return g.getDefault(d, k, module)
}
func (g *GlobalParams) GetValueName(name string) (string, error) {
	if g == nil {
		return "", ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return "", e
	}
	module, k, e := g.normalizeName(name)
	if e != nil {
		return "", e
	}
	return g.getValue(module, k)
}
func (g *GlobalParams) GetValueSymbol(name Symbol) (string, error) {
	if g == nil {
		return "", ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return "", e
	}
	module, k, e := g.normalizeSymbol(name)
	if e != nil {
		return "", e
	}
	return g.getValue(module, k)
}

// GetRef is the actual borrowed const-like alias, not a snapshot. Callers must
// neither Del nor mutate this Ref, and the registry must outlive its use.
func (g *GlobalParams) GetRef() (*ParamsRef, error) {
	if g == nil {
		return nil, ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return nil, e
	}
	return &g.globalParams, nil
}
