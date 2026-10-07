// Source: src/model/model_core.h:31-45,70-76; src/model/model.cpp:28-40,
// 60-66,111-150; src/model/model.h:26-36,48-56;
// src/tactic/model_converter.cpp:176-181.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package model

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// Initial decl2finterp storage. Function registration/interpretation operations
// remain unported; only empty source member construction/destruction is used
// by the current constant/Boolean path. No function values are fabricated.
type modelFunctionCell struct {
	declaration ast.ASTHandle
	hash, index uint32
	state       uint8
}
type modelFunctionMap struct{ Cells []modelFunctionCell }

func (t *modelFunctionMap) initialize(m *ast.ASTManager) error {
	if e := m.Budget().Reserve(8 * uint64(reflect.TypeOf(modelFunctionCell{}).Size())); e != nil {
		return e
	}
	t.Cells = make([]modelFunctionCell, 8)
	return nil
}
func (t *modelFunctionMap) empty() error {
	for _, cell := range t.Cells {
		if cell.state != modelMapFree {
			return errors.New("model: function interpretation operations remain unported")
		}
	}
	return nil
}
func (t *modelFunctionMap) close(m *ast.ASTManager) {
	m.Budget().Release(uint64(len(t.Cells)) * uint64(reflect.TypeOf(modelFunctionCell{}).Size()))
	t.Cells = nil
}

type modelUniverseCell struct {
	sort        ast.ASTHandle
	hash, index uint32
	used        bool
}
type modelUniverseMap struct {
	Cells []modelUniverseCell
	size  uint32
}

func (t *modelUniverseMap) resize(m *ast.ASTManager, n int) error {
	stride := uint64(reflect.TypeOf(modelUniverseCell{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := m.Budget().Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]modelUniverseCell, n)
	for _, cell := range t.Cells {
		if cell.used {
			i := int(cell.hash) & (n - 1)
			for next[i].used {
				i = (i + 1) & (n - 1)
			}
			next[i] = cell
		}
	}
	m.Budget().Release(uint64(len(t.Cells)) * stride)
	t.Cells = next
	return nil
}
func (t *modelUniverseMap) find(sort ast.ASTHandle, hash uint32) int {
	if len(t.Cells) == 0 {
		return -1
	}
	mask := len(t.Cells) - 1
	for i := int(hash) & mask; t.Cells[i].used; i = (i + 1) & mask {
		if t.Cells[i].sort == sort && t.Cells[i].hash == hash {
			return i
		}
	}
	return -1
}
func (t *modelUniverseMap) insert(m *ast.ASTManager, sort ast.ASTHandle, hash uint32) (int, error) {
	if len(t.Cells) == 0 {
		if e := t.resize(m, 8); e != nil {
			return -1, e
		}
	} else if uint64(t.size)*4 > uint64(len(t.Cells))*3 {
		if len(t.Cells) > int(^uint(0)>>1)/2 {
			return -1, arena.ErrBounds
		}
		if e := t.resize(m, len(t.Cells)*2); e != nil {
			return -1, e
		}
	}
	mask := len(t.Cells) - 1
	i := int(hash) & mask
	for t.Cells[i].used {
		if t.Cells[i].sort == sort && t.Cells[i].hash == hash {
			return i, nil
		}
		i = (i + 1) & mask
	}
	t.Cells[i] = modelUniverseCell{sort: sort, hash: hash, used: true}
	t.size++
	return i, nil
}
func (t *modelUniverseMap) close(m *ast.ASTManager) {
	m.Budget().Release(uint64(len(t.Cells)) * uint64(reflect.TypeOf(modelUniverseCell{}).Size()))
	t.Cells = nil
	t.size = 0
}

type modelUniverse struct {
	values     modelDeclarationVector
	sortOwned  bool
	ownerBytes uint64
}
type Model struct {
	self      *Model
	CoreField ModelCore
	sorts     modelDeclarationVector
	Universes modelUniverseMap
	// Scalar table indices resolve cold source ptr_vector<expr> owners here.
	directory  []*modelUniverse
	ownerBytes uint64
	Closed     bool
}

func NewModel(m *ast.ASTManager) (md *Model, err error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err = m.ProofsEnabled(); err != nil {
		return nil, err
	}
	bytes := uint64(reflect.TypeOf(Model{}).Size())
	if err = m.Budget().Reserve(bytes); err != nil {
		return nil, err
	}
	md = &Model{ownerBytes: bytes}
	md.self = md
	md.CoreField.manager = m
	md.CoreField.self = &md.CoreField
	md.CoreField.Derived = md
	defer func() {
		if err != nil {
			err = errors.Join(err, md.Close())
			md = nil
		}
	}()
	if err = md.CoreField.Constants.resize(m, 8); err != nil {
		return md, err
	}
	if err = md.CoreField.Functions.initialize(m); err != nil {
		return md, err
	}
	if err = md.Universes.resize(m, 8); err != nil {
		return md, err
	}
	return md, nil
}
func (md *Model) open() error {
	if md == nil || md.self != md {
		return ast.ErrASTHandle
	}
	if md.Closed {
		return arena.ErrClosed
	}
	return md.CoreField.open()
}
func (md *Model) sortHash(s ast.ASTHandle) (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	kind, e := md.CoreField.manager.GetKind(s)
	if e != nil {
		return 0, e
	}
	if kind != ast.ASTSort {
		return 0, ast.ErrASTDomain
	}
	return md.CoreField.manager.GetHash(s)
}
func (md *Model) NumUninterpretedSorts() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.sorts.Slice())), nil
}
func (md *Model) UninterpretedSort(i uint32) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.sorts.Slice())) {
		return ast.ASTHandle{}, arena.ErrBounds
	}
	return md.sorts.Slice()[i], nil
}
func (md *Model) HasUninterpretedSort(s ast.ASTHandle) (bool, error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return false, e
	}
	i := md.Universes.find(s, hash)
	return i >= 0 && md.Universes.Cells[i].index != 0, nil
}
func (md *Model) Universe(s ast.ASTHandle) ([]ast.ASTHandle, error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return nil, e
	}
	i := md.Universes.find(s, hash)
	if i < 0 || md.Universes.Cells[i].index == 0 {
		return nil, ast.ErrASTDomain
	}
	return md.directory[md.Universes.Cells[i].index-1].values.Slice(), nil
}
func (md *Model) RegisterUniverse(s ast.ASTHandle, values []ast.ASTHandle) (err error) {
	hash, e := md.sortHash(s)
	if e != nil {
		return e
	}
	m := md.CoreField.manager
	// Reject invalid Go handles before entering the source ownership sequence.
	for _, v := range values {
		kind, e := m.GetKind(v)
		if e != nil {
			return e
		}
		if kind != ast.ASTApp && kind != ast.ASTVar && kind != ast.ASTQuantifier {
			return ast.ErrASTDomain
		}
	}
	i, e := md.Universes.insert(m, s, hash)
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
	index := md.Universes.Cells[i].index
	if index == 0 {
		if e = md.sorts.push(m, s); e != nil {
			return e
		}
		if e = m.IncRef(s); e != nil {
			return e
		}
		bytes := uint64(reflect.TypeOf(modelUniverse{}).Size())
		if e = m.Budget().Reserve(bytes); e != nil {
			return errors.Join(e, m.DecRef(s))
		}
		u := &modelUniverse{sortOwned: true, ownerBytes: bytes}
		// Directory metadata provides the Go counterpart of the allocated vector
		// pointer. It is charged separately from source pointer-free map records.
		if len(md.directory) == cap(md.directory) {
			stride := uint64(reflect.TypeOf((*modelUniverse)(nil)).Size())
			n, e := util.SourceVectorCapacity(cap(md.directory), stride)
			if e != nil {
				m.Budget().Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
			if e = m.Budget().Reserve(uint64(n) * stride); e != nil {
				m.Budget().Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
			next := make([]*modelUniverse, len(md.directory), n)
			copy(next, md.directory)
			m.Budget().Release(uint64(cap(md.directory)) * stride)
			md.directory = next
		}
		for _, v := range values {
			if e = u.values.push(m, v); e != nil {
				u.values.close(m)
				m.Budget().Release(bytes)
				return errors.Join(e, m.DecRef(s))
			}
		}
		md.directory = append(md.directory, u)
		md.Universes.Cells[i].index = uint32(len(md.directory))
		published = true
	} else {
		u := md.directory[index-1]
		for _, old := range u.values.Slice() {
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
func (md *Model) CopyUniverseInterpretations(source *Model) error {
	if e := md.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	if md.CoreField.manager != source.CoreField.manager {
		return ast.ErrASTHandle
	}
	for _, cell := range source.Universes.Cells {
		if cell.used && cell.index != 0 {
			if e := md.RegisterUniverse(cell.sort, source.directory[cell.index-1].values.Slice()); e != nil {
				return e
			}
		}
	}
	return nil
}
func (md *Model) Close() error {
	if md == nil || md.self != md {
		return ast.ErrASTHandle
	}
	if md.Closed {
		return nil
	}
	m := md.CoreField.manager
	for i := range md.Universes.Cells {
		cell := &md.Universes.Cells[i]
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
		for j, v := range u.values.Slice() {
			if !v.IsNull() {
				if e := m.DecRef(v); e != nil {
					return e
				}
				u.values.Slice()[j] = ast.ASTHandle{}
			}
		}
		u.values.close(m)
		m.Budget().Release(u.ownerBytes)
		u.ownerBytes = 0
	}
	md.Universes.close(m)
	md.sorts.close(m)
	m.Budget().Release(uint64(cap(md.directory)) * uint64(reflect.TypeOf((*modelUniverse)(nil)).Size()))
	md.directory = nil
	if e := md.CoreField.closeCoreMembers(); e != nil {
		return e
	}
	m.Budget().Release(md.ownerBytes)
	md.ownerBytes = 0
	md.Closed = true
	return nil
}
