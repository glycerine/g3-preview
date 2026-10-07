// Source: src/model/model_core.h:31-45,70-76; src/model/model.cpp:28-40,
// 60-66,111-150; src/model/model.h:26-36,48-56;
// src/tactic/model_converter.cpp:176-181.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Initial decl2finterp storage. Function registration/interpretation operations
// remain unported; only empty source member construction/destruction is used
// by the current constant/Boolean path. No function values are fabricated.
type modelFunctionCell struct {
	declaration ASTHandle
	hash, index uint32
	state       uint8
}
type modelFunctionMap struct{ cells []modelFunctionCell }

func (t *modelFunctionMap) initialize(m *ASTManager) error {
	if e := m.budget.Reserve(8 * uint64(reflect.TypeOf(modelFunctionCell{}).Size())); e != nil {
		return e
	}
	t.cells = make([]modelFunctionCell, 8)
	return nil
}
func (t *modelFunctionMap) empty() error {
	for _, cell := range t.cells {
		if cell.state != modelMapFree {
			return errors.New("model: function interpretation operations remain unported")
		}
	}
	return nil
}
func (t *modelFunctionMap) close(m *ASTManager) {
	m.budget.Release(uint64(len(t.cells)) * uint64(reflect.TypeOf(modelFunctionCell{}).Size()))
	t.cells = nil
}

type modelUniverseCell struct {
	sort        ASTHandle
	hash, index uint32
	used        bool
}
type modelUniverseMap struct {
	cells []modelUniverseCell
	size  uint32
}

func (t *modelUniverseMap) resize(m *ASTManager, n int) error {
	stride := uint64(reflect.TypeOf(modelUniverseCell{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := m.budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]modelUniverseCell, n)
	for _, cell := range t.cells {
		if cell.used {
			i := int(cell.hash) & (n - 1)
			for next[i].used {
				i = (i + 1) & (n - 1)
			}
			next[i] = cell
		}
	}
	m.budget.Release(uint64(len(t.cells)) * stride)
	t.cells = next
	return nil
}
func (t *modelUniverseMap) find(sort ASTHandle, hash uint32) int {
	if len(t.cells) == 0 {
		return -1
	}
	mask := len(t.cells) - 1
	for i := int(hash) & mask; t.cells[i].used; i = (i + 1) & mask {
		if t.cells[i].sort == sort && t.cells[i].hash == hash {
			return i
		}
	}
	return -1
}
func (t *modelUniverseMap) insert(m *ASTManager, sort ASTHandle, hash uint32) (int, error) {
	if len(t.cells) == 0 {
		if e := t.resize(m, 8); e != nil {
			return -1, e
		}
	} else if uint64(t.size)*4 > uint64(len(t.cells))*3 {
		if len(t.cells) > int(^uint(0)>>1)/2 {
			return -1, arena.ErrBounds
		}
		if e := t.resize(m, len(t.cells)*2); e != nil {
			return -1, e
		}
	}
	mask := len(t.cells) - 1
	i := int(hash) & mask
	for t.cells[i].used {
		if t.cells[i].sort == sort && t.cells[i].hash == hash {
			return i, nil
		}
		i = (i + 1) & mask
	}
	t.cells[i] = modelUniverseCell{sort: sort, hash: hash, used: true}
	t.size++
	return i, nil
}
func (t *modelUniverseMap) close(m *ASTManager) {
	m.budget.Release(uint64(len(t.cells)) * uint64(reflect.TypeOf(modelUniverseCell{}).Size()))
	t.cells = nil
	t.size = 0
}

type modelUniverse struct {
	values     modelDeclarationVector
	sortOwned  bool
	ownerBytes uint64
}
type solverModel struct {
	self      *solverModel
	core      modelCore
	sorts     modelDeclarationVector
	universes modelUniverseMap
	// Scalar table indices resolve cold source ptr_vector<expr> owners here.
	directory  []*modelUniverse
	ownerBytes uint64
	closed     bool
}

func newSolverModel(m *ASTManager) (md *solverModel, err error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if _, err = m.ProofsEnabled(); err != nil {
		return nil, err
	}
	bytes := uint64(reflect.TypeOf(solverModel{}).Size())
	if err = m.budget.Reserve(bytes); err != nil {
		return nil, err
	}
	md = &solverModel{ownerBytes: bytes}
	md.self = md
	md.core.manager = m
	md.core.self = &md.core
	md.core.derived = md
	defer func() {
		if err != nil {
			err = errors.Join(err, md.Close())
			md = nil
		}
	}()
	if err = md.core.constants.resize(m, 8); err != nil {
		return md, err
	}
	if err = md.core.functions.initialize(m); err != nil {
		return md, err
	}
	if err = md.universes.resize(m, 8); err != nil {
		return md, err
	}
	return md, nil
}
func (md *solverModel) open() error {
	if md == nil || md.self != md {
		return ErrASTHandle
	}
	if md.closed {
		return arena.ErrClosed
	}
	return md.core.open()
}
func (md *solverModel) sortHash(s ASTHandle) (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	kind, e := md.core.manager.GetKind(s)
	if e != nil {
		return 0, e
	}
	if kind != ASTSort {
		return 0, ErrASTDomain
	}
	return md.core.manager.GetHash(s)
}
func (md *solverModel) NumUninterpretedSorts() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.sorts.values)), nil
}
func (md *solverModel) UninterpretedSort(i uint32) (ASTHandle, error) {
	if e := md.open(); e != nil {
		return ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.sorts.values)) {
		return ASTHandle{}, arena.ErrBounds
	}
	return md.sorts.values[i], nil
}
func (md *solverModel) HasUninterpretedSort(s ASTHandle) (bool, error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return false, e
	}
	i := md.universes.find(s, hash)
	return i >= 0 && md.universes.cells[i].index != 0, nil
}
func (md *solverModel) Universe(s ASTHandle) ([]ASTHandle, error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return nil, e
	}
	i := md.universes.find(s, hash)
	if i < 0 || md.universes.cells[i].index == 0 {
		return nil, ErrASTDomain
	}
	return md.directory[md.universes.cells[i].index-1].values.values, nil
}
func (md *solverModel) RegisterUniverse(s ASTHandle, values []ASTHandle) (err error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return e
	}
	m := md.core.manager
	// Reject invalid Go handles before entering the source ownership sequence.
	for _, v := range values {
		kind, e := m.GetKind(v)
		if e != nil {
			return e
		}
		if kind != ASTApp && kind != ASTVar && kind != ASTQuantifier {
			return ErrASTDomain
		}
	}
	i, e := md.universes.insert(m, s, hash)
	if e != nil {
		return e
	}
	acquired := 0
	published := false
	defer func() {
		if err != nil && !published {
			for j := 0; j < acquired; j++ {
				err = errors.Join(err, m.DecRef(values[j]))
			}
		}
	}()
	for _, v := range values {
		if e = m.IncRef(v); e != nil {
			return e
		}
		acquired++
	}
	index := md.universes.cells[i].index
	if index == 0 {
		if e = md.sorts.push(m, s); e != nil {
			return e
		}
		if e = m.IncRef(s); e != nil {
			return e
		}
		bytes := uint64(reflect.TypeOf(modelUniverse{}).Size())
		if e = m.budget.Reserve(bytes); e != nil {
			return errors.Join(e, m.DecRef(s))
		}
		u := &modelUniverse{sortOwned: true, ownerBytes: bytes}
		// Directory metadata provides the Go counterpart of the allocated vector
		// pointer. It is charged separately from source pointer-free map records.
		if len(md.directory) == cap(md.directory) {
			stride := uint64(reflect.TypeOf((*modelUniverse)(nil)).Size())
			n, e := satGoalCapacity(cap(md.directory), stride)
			if e != nil {
				m.budget.Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
			if e = m.budget.Reserve(uint64(n) * stride); e != nil {
				m.budget.Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
			next := make([]*modelUniverse, len(md.directory), n)
			copy(next, md.directory)
			m.budget.Release(uint64(cap(md.directory)) * stride)
			md.directory = next
		}
		for _, v := range values {
			if e = u.values.push(m, v); e != nil {
				u.values.close(m)
				m.budget.Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
		}
		md.directory = append(md.directory, u)
		md.universes.cells[i].index = uint32(len(md.directory))
		published = true
	} else {
		u := md.directory[index-1]
		for _, old := range u.values.values {
			if e = m.DecRef(old); e != nil {
				return e
			}
		}
		// Literal source update appends without resetting the old physical vector.
		// Preserve that behavior; do not invent extra roots for its stale prefix.
		for _, v := range values {
			if e = u.values.push(m, v); e != nil {
				return e
			}
		}
		published = true
	}
	return nil
}
func (md *solverModel) CopyUniverseInterpretations(source *solverModel) error {
	if e := md.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	if md.core.manager != source.core.manager {
		return ErrASTHandle
	}
	for _, cell := range source.universes.cells {
		if cell.used && cell.index != 0 {
			if e := md.RegisterUniverse(cell.sort, source.directory[cell.index-1].values.values); e != nil {
				return e
			}
		}
	}
	return nil
}
func (md *solverModel) Close() error {
	if md == nil || md.self != md {
		return ErrASTHandle
	}
	if md.closed {
		return nil
	}
	m := md.core.manager
	for i := range md.universes.cells {
		cell := &md.universes.cells[i]
		if !cell.used || cell.index == 0 {
			continue
		}
		u := md.directory[cell.index-1]
		if u.sortOwned {
			if e := m.DecRef(cell.sort); e != nil {
				return e
			}
			u.sortOwned = false
		}
		for j, v := range u.values.values {
			if !v.IsNull() {
				if e := m.DecRef(v); e != nil {
					return e
				}
				u.values.values[j] = ASTHandle{}
			}
		}
		u.values.close(m)
		m.budget.Release(u.ownerBytes)
		u.ownerBytes = 0
	}
	md.universes.close(m)
	md.sorts.close(m)
	m.budget.Release(uint64(cap(md.directory)) * uint64(reflect.TypeOf((*modelUniverse)(nil)).Size()))
	md.directory = nil
	if e := md.core.closeCoreMembers(); e != nil {
		return e
	}
	m.budget.Release(md.ownerBytes)
	md.ownerBytes = 0
	md.closed = true
	return nil
}

func tacticModelConverterToModel(m *ASTManager, c tacticModelConverter, out *tacticModelRef) (err error) {
	if c == nil {
		return nil
	}
	md, err := newSolverModel(m)
	if err != nil {
		return err
	}
	if err = out.Set(&md.core); err != nil {
		return errors.Join(err, md.Close())
	}
	return c.ApplyModel(out, 0)
}
