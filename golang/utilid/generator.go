// Source: src/util/id_gen.h; src/util/vector.h. Licensed under the MIT license.
// Translation of src/util/id_gen.h and unsigned vector.h specialization.
package utilid

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
)

var ErrAssertion = errors.New("g3: ID vector source assertion")

var ErrOwner = errors.New("g3: invalid or closed ID generator owner")
var ErrInterrupted = errors.New("g3: interrupted ID vector publication requires owning cleanup")

// Fresh and PopIndex are the shared source counter and LIFO algorithms used by
// the raw specialization and the legacy Go-typed main-package storage adapter.
func Fresh(next *uint32) uint32 { v := *next; *next = *next + 1; return v }
func PopIndex(size uint32) (index uint32, ok bool) {
	if size == 0 {
		return 0, false
	}
	return size - 1, true
}

// All arithmetic here is source unsigned (32 bits), including byte extents.
func VectorGrowth(old uint32) (capacity, bytes uint32, err error) {
	if old == 0 {
		return 2, 16, nil
	}
	n := (3*old + 1) >> 1
	nb := 4*n + 8
	ob := 4*old + 8
	if n <= old || nb <= ob {
		return 0, 0, memory.Failure{Kind: memory.FailureVectorOverflow}
	}
	return n, nb, nil
}

// Generator is embedded cold metadata. The enclosing owner charges its header.
// IDs live exclusively in the actual unsigned vector bytes, not a Go mirror.
// pending is the initial allocation's real lexical mem before m_data publication.
// closed is an owning cleanup guard, not a C++ exception recovery mechanism.
type Generator struct {
	self                *Generator
	runtime             *memory.Runtime
	next                uint32
	raw, pending        memory.Block
	interrupted, closed bool
	partialRaw          bool
}

func (g *Generator) Init(start uint32, r *memory.Runtime) error {
	if g == nil || r == nil || g.self != nil {
		return ErrOwner
	}
	if _, e := r.AllocationSize(); e != nil {
		return e
	}
	g.self = g
	g.runtime = r
	g.next = start
	return nil
}
func (g *Generator) ready() error {
	if g == nil || g.self != g || g.runtime == nil || g.closed {
		return ErrOwner
	}
	if g.interrupted {
		return ErrInterrupted
	}
	_, e := g.runtime.AllocationSize()
	return e
}
func (g *Generator) read(off uint64) (uint32, error) {
	var v uint32
	for i := uint64(0); i < 4; i++ {
		b, e := g.runtime.Read(g.raw, off+i)
		if e != nil {
			return 0, e
		}
		v |= uint32(b) << (8 * i)
	}
	return v, nil
}
func (g *Generator) write(h memory.Block, off uint64, v uint32) error {
	for i := uint64(0); i < 4; i++ {
		if e := g.runtime.Write(h, off+i, byte(v>>(8*i))); e != nil {
			return e
		}
	}
	return nil
}
func (g *Generator) Size() (uint32, error) {
	if e := g.ready(); e != nil {
		return 0, e
	}
	if g.raw == (memory.Block{}) {
		return 0, nil
	}
	return g.read(4)
}
func (g *Generator) Capacity() (uint32, error) {
	if e := g.ready(); e != nil {
		return 0, e
	}
	if g.raw == (memory.Block{}) {
		return 0, nil
	}
	return g.read(0)
}
func (g *Generator) Range() (uint32, error) {
	if e := g.ready(); e != nil {
		return 0, e
	}
	return g.next, nil
}
func (g *Generator) RawRoots() [2]memory.Block {
	if g == nil || g.self != g {
		return [2]memory.Block{}
	}
	return [2]memory.Block{g.raw, g.pending}
}
func (g *Generator) Next() (uint32, error) {
	n, e := g.Size()
	if e != nil {
		return 0, e
	}
	i, ok := PopIndex(n)
	if !ok {
		return Fresh(&g.next), nil
	}
	v, e := g.read(8 + 4*uint64(i))
	if e != nil {
		return 0, e
	}
	if e = g.write(g.raw, 4, i); e != nil {
		g.interrupted = true
		return 0, e
	}
	return v, nil
}
func (g *Generator) grow() error {
	c, e := g.Capacity()
	if e != nil {
		return e
	}
	// Source distinguishes m_data==null from malformed nonnull cap0.
	if g.raw != (memory.Block{}) && c == 0 {
		if sourceDebug {
			return ErrAssertion
		}
		return memory.Failure{Kind: memory.FailureVectorOverflow}
	}
	nc, nb, e := VectorGrowth(c)
	if e != nil {
		return e
	}
	if g.raw == (memory.Block{}) {
		g.pending, e = g.runtime.Allocate(uint64(nb))
		if e != nil {
			g.interrupted = g.pending != (memory.Block{})
			return e
		}
		if e = g.write(g.pending, 0, nc); e != nil {
			g.interrupted = true
			return e
		}
		if e = g.write(g.pending, 4, 0); e != nil {
			g.interrupted = true
			return e
		}
		g.raw = g.pending
		g.pending = memory.Block{}
		return nil
	}
	old := g.raw
	if e = g.runtime.Reallocate(&g.raw, uint64(nb)); e != nil {
		g.interrupted = g.raw != old
		if g.interrupted {
			// Actual changed token with wrapped SIZE_T extent cannot contain header.
			g.partialRaw = g.runtime.SizeTBits() == 32 && uint32(nb+4) < 4
		}
		return e
	}
	if e = g.write(g.raw, 0, nc); e != nil {
		g.interrupted = true
		return e
	}
	return nil
}
func (g *Generator) Recycle(id uint32) error {
	if e := g.ready(); e != nil {
		return e
	}
	if g.runtime.State().OutOfMemory {
		return nil
	}
	n, e := g.Size()
	if e != nil {
		return e
	}
	c, e := g.Capacity()
	if e != nil {
		return e
	}
	if n > c {
		return arena.ErrBounds
	}
	if n == c {
		if e = g.grow(); e != nil {
			return e
		}
	}
	if e = g.write(g.raw, 8+4*uint64(n), id); e != nil {
		g.interrupted = true
		return e
	}
	if e = g.write(g.raw, 4, n+1); e != nil {
		g.interrupted = true
		return e
	}
	return nil
}
func (g *Generator) Reset(start uint32) error {
	if e := g.ready(); e != nil {
		return e
	}
	g.next = start
	if g.raw == (memory.Block{}) {
		return nil
	}
	if e := g.write(g.raw, 4, 0); e != nil {
		g.interrupted = true
		return e
	}
	return nil
}

// Cleanup resumes only actual token releases. It changes the counter before
// vector finalization, as source cleanup does. Completed frees clear tokens.
func (g *Generator) Cleanup(start uint32) error {
	if g == nil || g.self != g || g.runtime == nil {
		return ErrOwner
	}
	g.next = start
	if g.pending != (memory.Block{}) {
		if e := g.runtime.Deallocate(&g.pending); e != nil {
			return e
		}
	}
	if g.raw != (memory.Block{}) {
		var e error
		if g.partialRaw {
			e = g.runtime.CleanupPartial(&g.raw)
		} else {
			e = g.runtime.Deallocate(&g.raw)
		}
		if e != nil {
			return e
		}
		g.partialRaw = false
	}
	g.interrupted = false
	return nil
}
func (g *Generator) CloseChecked() error {
	if g == nil || g.self != g {
		return ErrOwner
	}
	g.closed = true
	return g.Cleanup(g.next)
}

// SkipFree preserves source std::find's ascending scan on each candidate,
// including unsigned wrap. The accessor binds the owner's actual storage.
func SkipFree(next *uint32, size uint32, at func(uint32) (uint32, error)) (uint32, error) {
	for {
		found := false
		for i := uint32(0); i < size; i++ {
			v, e := at(i)
			if e != nil {
				return 0, e
			}
			if v == *next {
				found = true
				break
			}
		}
		if !found {
			return *next, nil
		}
		*next = *next + 1
	}
}
func (g *Generator) SetNextID(id uint32) (uint32, error) {
	if e := g.ready(); e != nil {
		return 0, e
	}
	g.next = id
	n, e := g.Size()
	if e != nil {
		return 0, e
	}
	return SkipFree(&g.next, n, func(i uint32) (uint32, error) { return g.read(8 + 4*uint64(i)) })
}

// ShowHash defines an empty free-list diagnostic profile rather than reproducing
// source &free_ids[0] undefined access. Nonempty hashing uses actual source bytes.
func (g *Generator) ShowHash() (uint32, error) {
	n, e := g.Size()
	if e != nil {
		return 0, e
	}
	var readError error
	h := utilhash.WordReader(n*4, 17, func(i uint32) uint32 {
		if readError != nil {
			return 0
		}
		v, e := g.read(8 + 4*uint64(i))
		if e != nil {
			readError = e
		}
		return v
	})
	if readError != nil {
		return 0, readError
	}
	return utilhash.UintPair(h, g.next), nil
}

// TextSink is borrowed cold output state; no interface is stored per ID.
type TextSink interface {
	WriteLiteral(string) error
	WriteUnsigned(uint32) error
}

func (g *Generator) DisplayFreeIDs(out TextSink) error {
	n, e := g.Size()
	if e != nil {
		return e
	}
	if out == nil {
		return arena.ErrConfig
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			if e = out.WriteLiteral(" "); e != nil {
				return e
			}
		}
		v, e := g.read(8 + 4*uint64(i))
		if e != nil {
			return e
		}
		if e = out.WriteUnsigned(v); e != nil {
			return e
		}
	}
	return nil
}

// SourceFinalize is the literal source vector destructor: only published m_data
// is deallocated. Initial lexical mem is outside that destructor's authority.
// Normal API entry stops before destructor actions; owning Cleanup remains
// available for pending locals, guarded headers and actual release progress.
func (g *Generator) SourceFinalize() error {
	if g == nil || g.self != g || g.runtime == nil {
		return ErrOwner
	}
	g.closed = true
	if g.raw != (memory.Block{}) {
		return g.runtime.Deallocate(&g.raw)
	}
	return nil
}
