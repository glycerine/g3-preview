// Source: src/ast/pp.cpp:22-132; src/ast/pp_params.pyg:1-23.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type astFormatPPPair struct {
	format astNodeRef
	indent uint32
}
type astFormatPPFrame struct {
	next                                                      *astFormatPPFrame
	owner                                                     *ASTManager
	parameters                                                smtModuleParamsOwner
	helper                                                    *smtModuleParamsFrame
	todo                                                      []astFormatPPPair
	size                                                      uint32
	space                                                     astNodeRef
	spaceAcquired, spaceConsumed, sourceDone, astContextEnded bool
	header                                                    uint64
	invocation                                                uint64
	bodyCause, sourceCause, ownCause                          error
}

func (f *astFormatPPFrame) bytes() uint64 {
	n := f.header + uint64(len(f.todo))*uint64(reflect.TypeOf(astFormatPPPair{}).Size())
	for h := f.parameters.helpers; h != nil; h = h.next {
		n += h.header
	}
	return n
}
func (f *astFormatPPFrame) push(r astNodeRef, indent uint32) error {
	if f.size == uint32(len(f.todo)) {
		capacity := uint64(len(f.todo))
		next := (capacity*3 + 1) / 2
		if next < 2 {
			next = 2
		}
		width := uint64(reflect.TypeOf(astFormatPPPair{}).Size())
		if next > math.MaxUint32 || next > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		if e := f.owner.budget.Reserve(next * width); e != nil {
			return e
		}
		a := make([]astFormatPPPair, int(next))
		for i := uint32(0); i < f.size; i++ {
			a[i] = f.todo[i]
		}
		f.owner.budget.Release(capacity * width)
		f.todo = a
	}
	f.todo[f.size] = astFormatPPPair{r, indent}
	f.size++
	return nil
}
func (m *ASTManager) ppFramesBytes() uint64 {
	var n uint64
	for f := m.ppFrames; f != nil; f = f.next {
		n += f.bytes()
	}
	return n
}
func (m *ASTManager) ppAssertFormat(r astNodeRef, g *GlobalParamsService, line int32) error {
	// SASSERT switch is reached before either actual expression operand.
	if !astSourceDebug {
		return nil
	}
	enabled, e := g.namespaceAssertionsEnabled()
	if e != nil {
		return e
	}
	if !enabled {
		return nil
	}
	if e = m.validNode(r); e != nil {
		return e
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return ErrASTDomain
	}
	id, _, e := m.declIdentity(m.appAt(r.slot).decl)
	if e != nil {
		return e
	}
	symbol, e := m.symbols.Intern("format")
	if e != nil {
		return e
	}
	expected, e := m.families.GetID(symbol)
	if e != nil {
		return e
	}
	if id != expected {
		return g.namespaceAssertionFailure("src/ast/pp.cpp", line, "f->get_family_id() == fm(m).get_family_id(\"format\")")
	}
	return nil
}
func (m *ASTManager) formatKind(r astNodeRef) (int32, error) {
	if e := m.validNode(r); e != nil {
		return 0, e
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return 0, ErrASTDomain
	}
	_, kind, e := m.declIdentity(m.appAt(r.slot).decl)
	return kind, e
}
func (m *ASTManager) formatParameter(r astNodeRef) (*astParameterRecord, error) {
	if e := m.validNode(r); e != nil {
		return nil, e
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return nil, ErrASTDomain
	}
	d := m.appAt(r.slot).decl
	info, e := m.funcInfo(m.declAt(d.slot).info)
	if e != nil {
		return nil, e
	}
	token, e := m.infoParameter(info, 0)
	if e != nil {
		return nil, e
	}
	return m.parameter(token)
}
func (m *ASTManager) formatSymbolView(s Symbol) (ParamsStringView, error) {
	if s.owner != m.symbols.owner && !s.IsNumerical() && s.value != 0 {
		return ParamsStringView{}, ErrSymbolOwner
	}
	if s.IsNumerical() {
		if astSourceDebug {
			return ParamsStringView{}, ErrInvalidSymbol
		}
		return literalParamsStringView(""), nil
	}
	if s.value == 0 {
		return ParamsStringView{}, ErrInvalidSymbol
	}
	return ParamsStringView{region: m.symbols.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(s.value), Generation: s.generation}, offset: m.symbols.pointerWidth, mode: 3, present: true}, nil
}
func (m *ASTManager) formatSpaceToBreak(r astNodeRef, g *GlobalParamsService) (uint32, bool, error) {
	if e := m.ppAssertFormat(r, g, 25); e != nil {
		return 0, false, e
	}
	kind, e := m.formatKind(r)
	if e != nil {
		return 0, false, e
	}
	a := m.appAt(r.slot)
	switch kind {
	case ASTFormatString:
		p, e := m.formatParameter(r)
		if e != nil {
			return 0, false, e
		}
		if p.kind != ASTParameterSymbol {
			return 0, false, ErrASTDomain
		}
		view, e := m.formatSymbolView(p.symbol)
		if e != nil {
			return 0, false, e
		}
		n, e := view.length()
		return uint32(n), false, e
	case ASTFormatChoice, ASTFormatIndent:
		child, e := m.readInternal(a.args, 0)
		if e != nil {
			return 0, false, e
		}
		return m.formatSpaceToBreak(child, g)
	case ASTFormatCompose:
		var n uint32
		for i := uint32(0); i < a.count; i++ {
			child, e := m.readInternal(a.args, i)
			if e != nil {
				return 0, false, e
			}
			size, line, e := m.formatSpaceToBreak(child, g)
			if e != nil {
				return 0, false, e
			}
			n += size
			if line {
				return n, true, nil
			}
		}
		return n, false, nil
	case ASTFormatLineBreak, ASTFormatLineBreakExt:
		return 0, true, nil
	default:
		return 0, false, nil
	}
}
func (f *astFormatPPFrame) sourceDestroy(body error) error {
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	if f.sourceCause != nil {
		return errors.Join(body, f.sourceCause)
	}
	m := f.owner
	if f.spaceAcquired && !f.spaceConsumed {
		consumed, e := m.decNode(f.space)
		if consumed {
			f.spaceConsumed = true
		}
		if e != nil {
			f.sourceCause = e
			return errors.Join(body, e)
		}
	}
	m.budget.Release(uint64(len(f.todo)) * uint64(reflect.TypeOf(astFormatPPPair{}).Size()))
	f.todo = nil
	if f.helper != nil {
		f.parameters.running = true
		e := f.parameters.finishHelper(f.helper, body)
		f.parameters.running = false
		if e != nil && f.helper.destructorCause != nil {
			f.sourceCause = f.helper.destructorCause
			return e
		}
	}
	f.sourceDone = true
	return body
}
func (m *ASTManager) releasePPFrames() error {
	for f := m.ppFrames; f != nil; f = f.next {
		if f.spaceAcquired && !f.spaceConsumed && !f.astContextEnded {
			consumed, e := m.decNode(f.space)
			if consumed {
				f.spaceConsumed = true
			}
			if e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
		}
		// This owning path is invoked only after the actual containing AST table
		// has ended: an already-consumed source child cannot be DEC'd again.
		m.budget.Release(uint64(len(f.todo)) * uint64(reflect.TypeOf(astFormatPPPair{}).Size()))
		f.todo = nil
		if e := f.parameters.closeHelpers(); e != nil {
			if f.ownCause == nil {
				f.ownCause = e
			}
			return e
		}
	}
	for f := m.ppFrames; f != nil; f = f.next {
		m.budget.Release(f.header)
		f.header = 0
	}
	m.ppFrames = nil
	return nil
}

func (m *ASTManager) PrintFormat(out *NumericTextStream, h ASTHandle, c *ParamsContext, p *ParamsRef, g *GlobalParamsService) (result error) {
	f, e := m.lockFormat()
	if e != nil {
		return e
	}
	defer f.unlockFormat()
	if out == nil || c == nil || c.self != c || g == nil || g.self != g || g.context != c || c.symbols != f.symbols || g.config.Budget != f.budget {
		return ErrParamsOwner
	}
	r, e := f.requireExpr(h)
	if e != nil {
		return e
	}
	if f.ppFrameSerial == math.MaxUint64 {
		return arena.ErrBounds
	}
	n := uint64(reflect.TypeOf(astFormatPPFrame{}).Size())
	if e = f.budget.Reserve(n); e != nil {
		return e
	}
	f.ppFrameSerial++
	frame := &astFormatPPFrame{owner: f, header: n, invocation: f.ppFrameSerial, next: f.ppFrames}
	f.ppFrames = frame
	frame.parameters = smtModuleParamsOwner{context: c, globals: g, budget: f.budget}
	frame.parameters.self = &frame.parameters
	defer func() {
		result = frame.sourceDestroy(result)
		if result == nil && frame.sourceDone {
			f.ppFrames = frame.next
			f.budget.Release(frame.header)
			frame.header = 0
		}
	}()
	frame.parameters.running = true
	frame.helper, e = frame.parameters.beginHelper(p, "pp")
	frame.parameters.running = false
	if e != nil {
		return e
	}
	frame.parameters.running = true
	maxWidth, e := frame.parameters.frameUInt(frame.helper, "max_width", 80)
	if e != nil {
		frame.parameters.running = false
		return e
	}
	maxRibbon, e := frame.parameters.frameUInt(frame.helper, "max_ribbon", 80)
	if e != nil {
		frame.parameters.running = false
		return e
	}
	maxLines, e := frame.parameters.frameUInt(frame.helper, "max_num_lines", math.MaxUint32)
	if e != nil {
		frame.parameters.running = false
		return e
	}
	maxIndent, e := frame.parameters.frameUInt(frame.helper, "max_indent", math.MaxUint32)
	if e != nil {
		frame.parameters.running = false
		return e
	}
	bounded, e := frame.parameters.frameBool(frame.helper, "bounded", false)
	if e != nil {
		frame.parameters.running = false
		return e
	}
	singleLine, e := frame.parameters.frameBool(frame.helper, "single_line", false)
	frame.parameters.running = false
	if e != nil {
		return e
	}
	var pos, ribbonPos, line uint32
	if e = frame.push(r, 0); e != nil {
		return e
	}
	symbol, e := f.symbols.Intern(" ")
	if e != nil {
		return e
	}
	parameter := astParameterRecord{kind: ASTParameterSymbol, symbol: symbol}
	frame.space, e = f.formatApp(ASTFormatString, &parameter, nil)
	if e != nil {
		return e
	}
	if e = f.incNode(frame.space); e != nil {
		return e
	}
	frame.spaceAcquired = true
	for frame.size != 0 {
		if line >= maxLines {
			return nil
		}
		pair := frame.todo[frame.size-1]
		r = pair.format
		indent := pair.indent
		frame.size--
		if e = f.ppAssertFormat(r, g, 84); e != nil {
			return e
		}
		kind, e := f.formatKind(r)
		if e != nil {
			return e
		}
		a := f.appAt(r.slot)
		switch kind {
		case ASTFormatString:
			if bounded && pos > maxWidth {
				break
			}
			parameter, e := f.formatParameter(r)
			if e != nil {
				return e
			}
			if parameter.kind != ASTParameterSymbol {
				return ErrASTDomain
			}
			view, e := f.formatSymbolView(parameter.symbol)
			if e != nil {
				return e
			}
			wide, e := view.length()
			if e != nil {
				return e
			}
			length := uint32(wide)
			if bounded && pos+length > maxWidth {
				if e = out.AppendCString("..."); e != nil {
					return e
				}
				break
			}
			pos += length
			ribbonPos += length
			// Symbol insertion is separate from its earlier strlen and inherits the
			// actual caller stream flags, including width consumption and badbit.
			if e = out.appendASTSymbol(f, parameter.symbol); e != nil {
				return e
			}
		case ASTFormatIndent:
			child, e := f.readInternal(a.args, 0)
			if e != nil {
				return e
			}
			parameter, e := f.formatParameter(r)
			if e != nil {
				return e
			}
			if parameter.kind != ASTParameterInt {
				return ErrASTDomain
			}
			next := indent + uint32(parameter.integer)
			if maxIndent < next {
				next = maxIndent
			}
			if e = frame.push(child, next); e != nil {
				return e
			}
		case ASTFormatCompose:
			i := a.count
			for i > 0 {
				i--
				child, e := f.readInternal(a.args, i)
				if e != nil {
					return e
				}
				if e = frame.push(child, indent); e != nil {
					return e
				}
			}
		case ASTFormatChoice:
			left := maxWidth - pos
			if maxRibbon-pos < left {
				left = maxRibbon - pos
			}
			spaceLeft := int32(left)
			var child astNodeRef
			if spaceLeft > 0 {
				first, e := f.readInternal(a.args, 0)
				if e != nil {
					return e
				}
				size, _, e := f.formatSpaceToBreak(first, g)
				if e != nil {
					return e
				}
				if size <= uint32(spaceLeft) {
					child = first
				} else {
					child, e = f.readInternal(a.args, 1)
					if e != nil {
						return e
					}
				}
			} else {
				child, e = f.readInternal(a.args, 1)
				if e != nil {
					return e
				}
			}
			if e = frame.push(child, indent); e != nil {
				return e
			}
		case ASTFormatLineBreak, ASTFormatLineBreakExt:
			if singleLine {
				if e = frame.push(frame.space, indent); e != nil {
					return e
				}
				break
			}
			pos = indent
			ribbonPos = 0
			line++
			if line < maxLines {
				if e = out.AppendCString("\n"); e != nil {
					return e
				}
				for i := uint32(0); i < indent; i++ {
					if e = out.AppendCString(" "); e != nil {
						return e
					}
				}
			} else {
				if e = out.AppendCString("...\n"); e != nil {
					return e
				}
			}
		}
	}
	_ = ribbonPos // actual source writes it but CHOICE reads pos for both limits.
	return nil
}

// Stamp the actual allocator child end at its reached Close, not by table size,
// root counts or a scan. Every recorded space is non-owning after this exact
// containing placement context ends; later owning metadata never DEC's it.
func (m *ASTManager) ppContextEnded() {
	for f := m.ppFrames; f != nil; f = f.next {
		f.astContextEnded = true
	}
}
