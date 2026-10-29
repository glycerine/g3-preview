package spacer

// Source: src/muz/spacer/spacer_context.h and spacer_context.cpp
// pred_transformer::frames.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type FrameTransformer interface {
	EnsureLevel(uint32) error
	AddLemmaCore(*Lemma) error
	IsInvariant(level uint32, formula ast.ASTHandle) (invariant bool, solverLevel uint32, err error)
}

type Frames struct {
	manager     *ast.ASTManager
	transformer FrameTransformer
	lemmas      []*Lemma
	size        uint32
	sorted      bool
	closed      bool
}

func NewFrames(manager *ast.ASTManager, transformer FrameTransformer) (*Frames, error) {
	if manager == nil || transformer == nil {
		return nil, ErrManagerOwner
	}
	return &Frames{manager: manager, transformer: transformer, sorted: true}, nil
}

func (f *Frames) open() error {
	if f == nil || f.closed || f.manager == nil || f.transformer == nil {
		return ErrManagerOwner
	}
	return nil
}

func (f *Frames) AddFrame() error {
	if err := f.open(); err != nil {
		return err
	}
	f.size++
	return nil
}

func (f *Frames) Size() (uint32, error) {
	if err := f.open(); err != nil {
		return 0, err
	}
	return f.size, nil
}

func (f *Frames) LemmaSize() (int, error) {
	if err := f.open(); err != nil {
		return 0, err
	}
	return len(f.lemmas), nil
}

func (f *Frames) less(left, right *Lemma) (bool, error) {
	l, err := left.Level()
	if err != nil {
		return false, err
	}
	r, err := right.Level()
	if err != nil {
		return false, err
	}
	if l != r {
		return l < r, nil
	}
	a, err := left.Body()
	if err != nil {
		return false, err
	}
	b, err := right.Body()
	if err != nil {
		return false, err
	}
	return f.manager.ASTLess(a, b)
}

func (f *Frames) sort() error {
	if f.sorted {
		return nil
	}
	for i := 1; i < len(f.lemmas); i++ {
		for j := i; j > 0; j-- {
			less, err := f.less(f.lemmas[j], f.lemmas[j-1])
			if err != nil {
				return err
			}
			if !less {
				break
			}
			f.lemmas[j-1], f.lemmas[j] = f.lemmas[j], f.lemmas[j-1]
		}
	}
	f.sorted = true
	return nil
}

func (f *Frames) AddLemma(lemma *Lemma) (added bool, resultErr error) {
	if err := f.open(); err != nil {
		return false, err
	}
	if lemma == nil || lemma.open() != nil || lemma.manager != f.manager {
		return false, ErrManagerOwner
	}
	body, err := lemma.Body()
	if err != nil {
		return false, err
	}
	for _, existing := range f.lemmas {
		other, bodyErr := existing.Body()
		if bodyErr != nil {
			return false, bodyErr
		}
		if other != body {
			continue
		}
		bindings, bindingErr := lemma.bindingHandles()
		if bindingErr != nil {
			return false, bindingErr
		}
		if bindingErr = existing.AddBindings(bindings); bindingErr != nil {
			return false, bindingErr
		}
		level, levelErr := lemma.Level()
		if levelErr != nil {
			return false, levelErr
		}
		existingLevel, levelErr := existing.Level()
		if levelErr != nil {
			return false, levelErr
		}
		if closeErr := lemma.Close(); closeErr != nil {
			return false, closeErr
		}
		if existingLevel >= level {
			return false, nil
		}
		if err = existing.setLevel(level); err != nil {
			return false, err
		}
		if err = f.transformer.AddLemmaCore(existing); err != nil {
			return false, err
		}
		f.sorted = false
		if err = f.sort(); err != nil {
			return false, err
		}
		return true, nil
	}
	f.lemmas = append(f.lemmas, lemma)
	f.sorted = false
	if err = f.transformer.AddLemmaCore(lemma); err != nil {
		f.lemmas = f.lemmas[:len(f.lemmas)-1]
		return false, err
	}
	return true, nil
}

func (f *Frames) FrameLemmas(level uint32) ([]ast.ASTHandle, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	var result []ast.ASTHandle
	for _, lemma := range f.lemmas {
		lemmaLevel, err := lemma.Level()
		if err != nil {
			return nil, err
		}
		if lemmaLevel == level {
			body, err := lemma.Body()
			if err != nil {
				return nil, err
			}
			result = append(result, body)
		}
	}
	return result, nil
}

func (f *Frames) FrameGEQLemmas(level uint32) ([]ast.ASTHandle, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	var result []ast.ASTHandle
	for _, lemma := range f.lemmas {
		lemmaLevel, err := lemma.Level()
		if err != nil {
			return nil, err
		}
		if lemmaLevel >= level {
			body, err := lemma.Body()
			if err != nil {
				return nil, err
			}
			result = append(result, body)
		}
	}
	return result, nil
}

func (f *Frames) PropagateToInfinity(level uint32) error {
	if err := f.open(); err != nil {
		return err
	}
	for _, lemma := range f.lemmas {
		lemmaLevel, err := lemma.Level()
		if err != nil {
			return err
		}
		if lemmaLevel >= level && lemmaLevel != InfinityLevel {
			if err = lemma.setLevel(InfinityLevel); err != nil {
				return err
			}
			if err = f.transformer.AddLemmaCore(lemma); err != nil {
				return err
			}
			f.sorted = false
		}
	}
	return nil
}

func (f *Frames) PropagateToNextLevel(level uint32) (bool, error) {
	if err := f.open(); err != nil {
		return false, err
	}
	if err := f.sort(); err != nil {
		return false, err
	}
	if len(f.lemmas) == 0 {
		return true, nil
	}
	target := level + 1
	if level == InfinityLevel {
		target = InfinityLevel
	}
	if err := f.transformer.EnsureLevel(target); err != nil {
		return false, err
	}
	all := true
	for _, lemma := range f.lemmas {
		lemmaLevel, err := lemma.Level()
		if err != nil {
			return false, err
		}
		if lemmaLevel < level {
			continue
		}
		if lemmaLevel > level {
			break
		}
		body, err := lemma.Body()
		if err != nil {
			return false, err
		}
		invariant, solverLevel, err := f.transformer.IsInvariant(target, body)
		if err != nil {
			return false, err
		}
		if !invariant {
			all = false
			continue
		}
		if err = lemma.setLevel(solverLevel); err != nil {
			return false, err
		}
		if err = f.transformer.AddLemmaCore(lemma); err != nil {
			return false, err
		}
		f.sorted = false
	}
	if !f.sorted {
		if err := f.sort(); err != nil {
			return false, err
		}
	}
	return all, nil
}

func (f *Frames) Close() (result error) {
	if f == nil {
		return ErrManagerOwner
	}
	if f.closed {
		return nil
	}
	if f.manager == nil {
		return ErrManagerOwner
	}
	for i := len(f.lemmas) - 1; i >= 0; i-- {
		result = errors.Join(result, f.lemmas[i].Close())
	}
	if result == nil {
		f.lemmas = nil
		f.manager = nil
		f.transformer = nil
		f.closed = true
	}
	return result
}
