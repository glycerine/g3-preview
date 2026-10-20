// Source: src/ast/static_features.cpp:163-365;
// src/ast/static_features.h:128-145.
package ast

func staticFeatureIncrement(values *[]uint32, fid FamilyID) {
	if fid < 0 {
		return
	}
	i := int(fid)
	if i >= len(*values) {
		*values = append(*values, make([]uint32, i+1-len(*values))...)
	}
	(*values)[i]++
}

func (f *StaticFeatures) markTheory(fid FamilyID) {
	// The manager installs the five source builtin families at IDs 0 through 4.
	if fid <= 4 {
		return
	}
	i := int(fid)
	if i >= len(f.theories) {
		f.theories = append(f.theories, make([]bool, i+1-len(f.theories))...)
	}
	if !f.theories[i] {
		f.theories[i] = true
		f.NumMarkedTheories++
	}
}

func (f *StaticFeatures) classifySort(sort ASTHandle) (FamilyID, error) {
	fid, err := f.manager.GetFamilyID(sort)
	if err != nil {
		return NullFamilyID, err
	}
	f.markTheory(fid)
	kind, err := f.manager.GetDeclKind(sort)
	if err != nil {
		return NullFamilyID, err
	}
	switch fid {
	case f.arithFID:
		if kind == 0 {
			f.HasReal = true
		} else if kind == 1 {
			f.HasInt = true
		}
	case f.bitVectorFID:
		if kind == 0 {
			f.HasBitVector = true
		}
	case f.fpaFID:
		if kind == 0 || kind == 1 {
			f.HasFloatingPoint = true
		}
	case f.arrayFID:
		if kind == 0 {
			f.HasArrays = true
		}
	case f.sequenceFID:
		if kind == 0 {
			if f.manager.sequencePlugin != nil && sort == f.manager.external(f.manager.sequencePlugin.stringSort) {
				f.HasString = true
			} else {
				f.HasNonStringSequence = true
			}
		}
	}
	return fid, nil
}

func (f *StaticFeatures) accountEquality(h ASTHandle, isEquality bool) error {
	if !isEquality {
		return nil
	}
	f.NumEqualities++
	lhs, err := f.manager.GetArg(h, 0)
	if err != nil {
		return err
	}
	sort, err := f.manager.GetSort(lhs)
	if err != nil {
		return err
	}
	fid, err := f.manager.GetFamilyID(sort)
	if err != nil {
		return err
	}
	if fid != NullFamilyID && fid != f.basicFID {
		staticFeatureIncrement(&f.NumTheoryEqualities, fid)
	}
	return nil
}

func (f *StaticFeatures) accountApplicationTheory(h ASTHandle, fid, resultFID FamilyID, isBool, isEquality, gate bool, args uint32) error {
	f.markTheory(fid)
	if fid != NullFamilyID && fid != f.basicFID {
		f.NumInterpretedExprs++
		if isBool {
			staticFeatureIncrement(&f.NumTheoryAtoms, fid)
		} else {
			staticFeatureIncrement(&f.NumTheoryTerms, fid)
		}
		if args == 0 {
			f.NumInterpretedConstants++
		}
	}
	if fid == NullFamilyID && args == 0 && resultFID != NullFamilyID && resultFID != f.basicFID {
		staticFeatureIncrement(&f.NumTheoryConstants, resultFID)
	}
	if isEquality || gate {
		return nil
	}
	for i := uint32(0); i < args; i++ {
		arg, err := f.manager.GetArg(h, i)
		if err != nil {
			return err
		}
		sort, err := f.manager.GetSort(arg)
		if err != nil {
			return err
		}
		argFID, err := f.manager.GetFamilyID(sort)
		if err != nil {
			return err
		}
		if argFID != NullFamilyID && argFID != fid {
			f.NumAliens++
			staticFeatureIncrement(&f.NumAliensPerFamily, argFID)
		}
	}
	return nil
}
