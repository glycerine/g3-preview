// Source: src/smt/theory_bv.cpp:75-137,210-220,263-297,1163-1185;
// theory_bv.h var_pos_occ and bit_atom.
package smtcore

import (
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
)

// Entries are stored in insertion order and traversed backwards to represent
// the source linked-list head. Repeated bits retain their distinct positions.
type smtBVBitOccurrence struct {
	variable SMTTheoryVar
	index    uint32
}

type smtBVBitOccurrenceTrail struct{ boolean int32 }

func (t *SMTBitVectorTheory) CreateVariable(node *SMTEnode) (SMTTheoryVar, error) {
	variable, err := t.SMTTheoryBase.CreateVariable(node)
	if err != nil {
		return variable, err
	}
	t.privateFind = append(t.privateFind, variable)
	t.privateSize = append(t.privateSize, 1)
	t.privateNext = append(t.privateNext, variable)
	t.privateTrail = append(t.privateTrail, smtBVPrivateTrail{kind: smtBVNewVariable})
	t.bitLiterals = append(t.bitLiterals, nil)
	t.watchedPosition = append(t.watchedPosition, 0)
	t.zeroOneBits = append(t.zeroOneBits, nil)
	return variable, nil
}

func (t *SMTBitVectorTheory) addBitOccurrence(boolean int32, variable SMTTheoryVar, index uint32) error {
	if boolean == SMTTrueBoolVar {
		return nil
	}
	if boolean < SMTFirstBoolVar || variable < 0 || int(variable) >= len(t.variables) {
		return ast.ErrASTDomain
	}
	occurrences, exists := t.bitAtoms[boolean]
	// Reentrant bit2bool internalization can encounter its own already
	// published occurrence. Source does not register the same pair twice.
	for _, entry := range occurrences {
		if entry.variable == variable && entry.index == index {
			return nil
		}
	}
	if !exists {
		t.bitAtoms[boolean] = nil
		t.bitAtomTrail = append(t.bitAtomTrail, boolean)
		t.privateTrail = append(t.privateTrail, smtBVPrivateTrail{kind: smtBVNewAtom, boolean: boolean})
	}
	t.bitAtoms[boolean] = append(occurrences, smtBVBitOccurrence{variable, index})
	t.bitOccurrenceTrail = append(t.bitOccurrenceTrail, smtBVBitOccurrenceTrail{boolean})
	t.privateTrail = append(t.privateTrail, smtBVPrivateTrail{kind: smtBVNewOccurrence, boolean: boolean})
	return nil
}

// Source add_bit publishes the literal before its occurrence. Each caller
// reaches this immediately after actual Boolean internalization of that bit.
func (t *SMTBitVectorTheory) registerBitLiteral(expression ast.ASTHandle, index uint32, literal SMTLiteral) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	node := s.context.enodes[expression]
	if node == nil {
		return ast.ErrASTDomain
	}
	variable := node.TheoryVar(t.FamilyID())
	if variable < 0 || int(variable) >= len(t.bitLiterals) {
		return ast.ErrASTDomain
	}
	bits := t.bitLiterals[variable]
	if uint64(index) < uint64(len(bits)) {
		if bits[index] != literal {
			return ast.ErrASTDomain
		}
		return nil
	}
	if uint64(index) != uint64(len(bits)) {
		return ast.ErrASTDomain
	}
	t.bitLiterals[variable] = append(bits, literal)
	if literal.Var() == SMTTrueBoolVar {
		t.zeroOneBits[variable] = append(t.zeroOneBits[variable], smtBVZeroOneBit{variable: variable, index: index, isTrue: literal == SMTTrueLiteral})
		return nil
	}
	if literal.Var() != SMTTrueBoolVar {
		data, err := s.BooleanData(literal.Var())
		if err != nil {
			return err
		}
		if !data.IsTheoryAtom() {
			if err = s.SetVariableTheory(literal.Var(), t.FamilyID()); err != nil {
				return err
			}
		}
	}
	if literal.Var() != SMTTrueBoolVar {
		if err = t.findNewBitDisequalities(variable, index, literal); err != nil {
			return err
		}
	}
	return t.addBitOccurrence(literal.Var(), variable, index)
}

// Source display_atoms orders bit atoms by Boolean variable, while each atom
// prints its newest occurrence first. Empty surviving atoms still print a row.
func (t *SMTBitVectorTheory) displayBitAtoms(out io.Writer) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	if out == nil {
		return ast.ErrASTHandle
	}
	if err = smtWriteTheoryText(out, "atoms:\n"); err != nil {
		return err
	}
	for boolean := int32(0); int(boolean) < len(s.boolExpr); boolean++ {
		entries, exists := t.bitAtoms[boolean]
		if !exists {
			continue
		}
		id, err := s.manager.GetASTID(s.boolExpr[boolean])
		if err != nil {
			return err
		}
		if err = smtWriteTheoryText(out, fmt.Sprintf("#%d ->", id)); err != nil {
			return err
		}
		for i := len(entries) - 1; i >= 0; i-- {
			entry := entries[i]
			node, err := t.Enode(entry.variable)
			if err != nil {
				return err
			}
			id, err := node.OwnerID()
			if err != nil {
				return err
			}
			if err = smtWriteTheoryText(out, fmt.Sprintf(" #%d[%d]", id, entry.index)); err != nil {
				return err
			}
		}
		if err = smtWriteTheoryText(out, "\n"); err != nil {
			return err
		}
	}
	return nil
}

// Source mk_bit2bool first internalizes its argument. That initialization may
// create this very Boolean atom; it must not be allocated ahead of earlier bits.
func (t *SMTBitVectorTheory) internalizeBitProjection(expression ast.ASTHandle) (bool, error) {
	s, err := t.State()
	if err != nil {
		return false, err
	}
	owner, err := s.manager.GetArg(expression, 0)
	if err != nil {
		return false, err
	}
	missing := s.context.enodes[owner] == nil
	if err = s.Internalize(owner, false); err != nil {
		return false, err
	}
	node := s.context.enodes[owner]
	if node == nil {
		return false, ast.ErrASTDomain
	}
	if node.TheoryVar(t.FamilyID()) == SMTNullTheoryVar {
		approximate, err := t.approximateTerm(owner)
		if err != nil {
			return false, err
		}
		if approximate {
			variable, found := s.BoolVar(expression)
			if !found {
				variable, err = s.MkBoolVar(expression)
				if err != nil {
					return false, err
				}
			}
			if err = s.SetVariableTheory(variable, t.FamilyID()); err != nil {
				return false, err
			}
			return true, nil // Preserve the existing approximate-term unknown result.
		}
		// Native get_var handles ITE enodes whose sort constraint was suppressed.
		if _, err = t.CreateVariable(node); err != nil {
			return false, err
		}
		bits, err := t.bitsFor(owner)
		if err != nil {
			return false, err
		}
		for i, bit := range bits {
			if err = s.Internalize(bit, true); err != nil {
				return false, err
			}
			literal, err := s.Literal(bit)
			if err != nil {
				return false, err
			}
			if err = t.registerBitLiteral(owner, uint32(i), literal); err != nil {
				return false, err
			}
		}
	}
	if _, found := s.BoolVar(expression); found {
		return true, nil
	}
	bits, err := t.bitsFor(owner)
	if err != nil {
		return false, err
	}
	index, err := t.bit2BoolIndex(expression)
	if err != nil {
		return false, err
	}
	if uint64(index) >= uint64(len(bits)) {
		return false, ast.ErrASTDomain
	}
	op, own, err := t.operation(owner)
	if err != nil {
		return false, err
	}
	if missing && own && op == smtBVOpNumeral {
		// Source creates all numeral projections in increasing index order on
		// this first visit; subsequent visits create only their requested atom.
		for i := range bits {
			atom, err := s.manager.MkBitVectorBit2Bool(uint32(i), owner)
			if err != nil {
				return false, err
			}
			if err = s.Internalize(atom, true); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	variable, err := s.MkBoolVar(expression)
	if err != nil {
		return false, err
	}
	if err = s.SetVariableTheory(variable, t.FamilyID()); err != nil {
		return false, err
	}
	if err = t.addBitOccurrence(variable, node.TheoryVar(t.FamilyID()), index); err != nil {
		return false, err
	}
	definition := bits[index]
	if definition != expression {
		if err = s.Internalize(definition, true); err != nil {
			return false, err
		}
		defined, err := s.Literal(definition)
		if err != nil {
			return false, err
		}
		literal := NewSMTLiteral(variable, false)
		if err = s.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal.Negated(), defined}); err != nil {
			return false, err
		}
		if err = s.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal, defined.Negated()}); err != nil {
			return false, err
		}
		t.statistics.bit2core++
		if s.relevancy != nil {
			if err = s.relevancy.AddDependency(expression, definition); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// Source find_new_diseq_axioms consumes the old linked head before add_bit
// prepends its own occurrence. Constants use the separate class summary path.
func (t *SMTBitVectorTheory) findNewBitDisequalities(variable SMTTheoryVar, index uint32, literal SMTLiteral) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	node, err := t.Enode(variable)
	if err != nil {
		return err
	}
	width, err := s.manager.BitVectorSize(node.owner)
	if err != nil {
		return err
	}
	old := t.bitAtoms[literal.Var()]
	for i := len(old) - 1; i >= 0; i-- {
		entry := old[i]
		if entry.index != index {
			continue
		}
		if entry.variable < 0 || int(entry.variable) >= len(t.bitLiterals) || uint64(index) >= uint64(len(t.bitLiterals[entry.variable])) {
			return ast.ErrASTDomain
		}
		if t.bitLiterals[entry.variable][index] != literal.Negated() {
			continue
		}
		other, err := t.Enode(entry.variable)
		if err != nil {
			return err
		}
		otherWidth, err := s.manager.BitVectorSize(other.owner)
		if err != nil {
			return err
		}
		if width != otherWidth {
			continue
		}
		if err = t.newStaticBitDisequality(variable, entry.variable, index); err != nil {
			return err
		}
	}
	return nil
}

// Source mk_new_diseq_axiom is also called by constant-summary merging.
func (t *SMTBitVectorTheory) newStaticBitDisequality(left, right SMTTheoryVar, index uint32) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	node, err := t.Enode(left)
	if err != nil {
		return err
	}
	other, err := t.Enode(right)
	if err != nil {
		return err
	}
	t.statistics.diseqStatic++
	equality, err := t.MkEqualityLiteral(node.owner, other.owner, true)
	if err != nil {
		return err
	}
	if err = s.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality.Negated()}); err != nil {
		return err
	}
	if s.relevancy != nil && s.relevancy.enabled {
		expression, err := s.BoolExpression(equality.Var())
		if err != nil {
			return err
		}
		if err = s.relevancy.AddPairDependency(node.owner, other.owner, expression); err != nil {
			return err
		}
	}
	return nil
}
