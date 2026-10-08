// Source: src/util/total_order.h:28-406.
package util

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type totalOrderCell struct {
	next, previous *totalOrderCell
	label          uint64
	data           int64
}

type totalOrderCore struct {
	cells map[int64]*totalOrderCell
	base  totalOrderCell
	size  uint32
}

func newTotalOrderCore() *totalOrderCore {
	order := &totalOrderCore{cells: make(map[int64]*totalOrderCell)}
	order.base.next = &order.base
	order.base.previous = &order.base
	return order
}

func (o *totalOrderCore) cell(value int64) *totalOrderCell {
	cell := o.cells[value]
	if cell == nil {
		panic("g3: total-order value is absent")
	}
	return cell
}

func (o *totalOrderCore) makeCell(value int64) *totalOrderCell {
	if o.cells[value] != nil {
		panic("g3: duplicate total-order value")
	}
	cell := &totalOrderCell{data: value}
	o.cells[value] = cell
	o.size++
	return cell
}

func (o *totalOrderCore) relative(cell *totalOrderCell) uint64 {
	return cell.label - o.base.label
}

func (o *totalOrderCore) nextRelative(cell *totalOrderCell) uint64 {
	if cell.next == &o.base {
		return math.MaxUint64
	}
	return o.relative(cell.next)
}

func (o *totalOrderCore) insertAfterCell(after, inserted *totalOrderCell) {
	a := o.relative(after)
	next := o.nextRelative(after)
	if next < 2 || a > next-2 {
		baseLabel := after.label
		idealGap := uint64(math.MaxUint64) / uint64(o.size)
		goalGap := idealGap / 32
		cell := after.next.next
		count := uint32(2)
		gap := (cell.label - baseLabel) / uint64(count)
		for count < o.size && gap < goalGap {
			count++
			cell = cell.next
			gap = (cell.label - baseLabel) / uint64(count)
		}
		if count == o.size {
			gap = idealGap
		}
		cell = after.next
		increment := gap
		for range count {
			cell.label = baseLabel + increment
			cell = cell.next
			increment += gap
		}
		a = o.relative(after)
		next = o.nextRelative(after)
	}
	inserted.label = a + (next-a)/2 + o.base.label
	inserted.next = after.next
	after.next.previous = inserted
	inserted.previous = after
	after.next = inserted
}

func (o *totalOrderCore) insert(value int64) {
	o.insertAfterCell(&o.base, o.makeCell(value))
}

func (o *totalOrderCore) insertAfter(after, value int64) {
	o.insertAfterCell(o.cell(after), o.makeCell(value))
}

func (o *totalOrderCore) removeCell(cell *totalOrderCell) {
	cell.previous.next = cell.next
	cell.next.previous = cell.previous
}

func (o *totalOrderCore) moveAfter(after, moved int64) {
	a, b := o.cell(after), o.cell(moved)
	if a == b {
		panic("g3: total-order move requires distinct values")
	}
	if a.next == b {
		return
	}
	o.removeCell(b)
	o.insertAfterCell(a, b)
}

func (o *totalOrderCore) moveBeginning(value int64) {
	cell := o.cell(value)
	if o.base.next == cell {
		return
	}
	o.removeCell(cell)
	o.insertAfterCell(&o.base, cell)
}

func (o *totalOrderCore) erase(value int64) {
	cell := o.cell(value)
	o.removeCell(cell)
	delete(o.cells, value)
	o.size--
}

func (o *totalOrderCore) contains(value int64) bool { return o.cells[value] != nil }
func (o *totalOrderCore) less(a, b int64) bool      { return o.relative(o.cell(a)) < o.relative(o.cell(b)) }
func (o *totalOrderCore) greater(a, b int64) bool {
	return o.relative(o.cell(a)) > o.relative(o.cell(b))
}
func (o *totalOrderCore) empty() bool  { return o.base.next == &o.base }
func (o *totalOrderCore) first() int64 { return o.nonempty().next.data }

func (o *totalOrderCore) nonempty() *totalOrderCell {
	if o.empty() {
		panic("g3: empty total order")
	}
	return &o.base
}

func (o *totalOrderCore) hasNext(value int64) bool { return o.cell(value).next != &o.base }
func (o *totalOrderCore) hasPred(value int64) bool { return o.cell(value).previous != &o.base }
func (o *totalOrderCore) next(value int64) int64 {
	cell := o.cell(value)
	if cell.next == &o.base {
		panic("g3: total-order value has no successor")
	}
	return cell.next.data
}
func (o *totalOrderCore) pred(value int64) int64 {
	cell := o.cell(value)
	if cell.previous == &o.base {
		panic("g3: total-order value has no predecessor")
	}
	return cell.previous.data
}

func (o *totalOrderCore) display(out io.Writer, detail bool) error {
	first := true
	for cell := o.base.next; cell != &o.base; cell = cell.next {
		if !first {
			if _, err := io.WriteString(out, " "); err != nil {
				return err
			}
		}
		first = false
		var text string
		if detail {
			text = fmt.Sprintf("%d:%d", cell.data, cell.label)
		} else {
			text = fmt.Sprintf("%d", cell.data)
		}
		if _, err := io.WriteString(out, text); err != nil {
			return err
		}
	}
	return nil
}

func (o *totalOrderCore) checkInvariant() bool {
	count := uint32(0)
	for cell := o.base.next; cell != &o.base; cell = cell.next {
		if cell.next != &o.base && o.relative(cell) >= o.relative(cell.next) {
			return false
		}
		if cell.next.previous != cell || cell.previous.next != cell || o.cells[cell.data] != cell {
			return false
		}
		count++
		if count > o.size {
			return false
		}
	}
	return count == o.size && uint32(len(o.cells)) == o.size
}

// UintTotalOrder is the source uint_total_order specialization.
type UintTotalOrder struct{ core *totalOrderCore }

func NewUintTotalOrder() *UintTotalOrder                    { return &UintTotalOrder{core: newTotalOrderCore()} }
func (o *UintTotalOrder) Contains(v uint32) bool            { return o.core.contains(int64(v)) }
func (o *UintTotalOrder) Insert(v uint32)                   { o.core.insert(int64(v)) }
func (o *UintTotalOrder) InsertAfter(a, b uint32)           { o.core.insertAfter(int64(a), int64(b)) }
func (o *UintTotalOrder) MoveBeginning(v uint32)            { o.core.moveBeginning(int64(v)) }
func (o *UintTotalOrder) MoveAfter(a, b uint32)             { o.core.moveAfter(int64(a), int64(b)) }
func (o *UintTotalOrder) Erase(v uint32)                    { o.core.erase(int64(v)) }
func (o *UintTotalOrder) LT(a, b uint32) bool               { return o.core.less(int64(a), int64(b)) }
func (o *UintTotalOrder) GT(a, b uint32) bool               { return o.core.greater(int64(a), int64(b)) }
func (o *UintTotalOrder) Empty() bool                       { return o.core.empty() }
func (o *UintTotalOrder) Size() uint32                      { return o.core.size }
func (o *UintTotalOrder) First() uint32                     { return uint32(o.core.first()) }
func (o *UintTotalOrder) HasNext(v uint32) bool             { return o.core.hasNext(int64(v)) }
func (o *UintTotalOrder) Next(v uint32) uint32              { return uint32(o.core.next(int64(v))) }
func (o *UintTotalOrder) HasPred(v uint32) bool             { return o.core.hasPred(int64(v)) }
func (o *UintTotalOrder) Pred(v uint32) uint32              { return uint32(o.core.pred(int64(v))) }
func (o *UintTotalOrder) Display(out io.Writer) error       { return o.core.display(out, false) }
func (o *UintTotalOrder) DisplayDetail(out io.Writer) error { return o.core.display(out, true) }
func (o *UintTotalOrder) CheckInvariant() bool              { return o.core.checkInvariant() }
func (o *UintTotalOrder) String() string                    { var b strings.Builder; _ = o.Display(&b); return b.String() }

// IntTotalOrder is the source int_total_order specialization.
type IntTotalOrder struct{ core *totalOrderCore }

func NewIntTotalOrder() *IntTotalOrder                     { return &IntTotalOrder{core: newTotalOrderCore()} }
func (o *IntTotalOrder) Contains(v int32) bool             { return o.core.contains(int64(v)) }
func (o *IntTotalOrder) Insert(v int32)                    { o.core.insert(int64(v)) }
func (o *IntTotalOrder) InsertAfter(a, b int32)            { o.core.insertAfter(int64(a), int64(b)) }
func (o *IntTotalOrder) MoveBeginning(v int32)             { o.core.moveBeginning(int64(v)) }
func (o *IntTotalOrder) MoveAfter(a, b int32)              { o.core.moveAfter(int64(a), int64(b)) }
func (o *IntTotalOrder) Erase(v int32)                     { o.core.erase(int64(v)) }
func (o *IntTotalOrder) LT(a, b int32) bool                { return o.core.less(int64(a), int64(b)) }
func (o *IntTotalOrder) GT(a, b int32) bool                { return o.core.greater(int64(a), int64(b)) }
func (o *IntTotalOrder) Empty() bool                       { return o.core.empty() }
func (o *IntTotalOrder) Size() uint32                      { return o.core.size }
func (o *IntTotalOrder) First() int32                      { return int32(o.core.first()) }
func (o *IntTotalOrder) HasNext(v int32) bool              { return o.core.hasNext(int64(v)) }
func (o *IntTotalOrder) Next(v int32) int32                { return int32(o.core.next(int64(v))) }
func (o *IntTotalOrder) HasPred(v int32) bool              { return o.core.hasPred(int64(v)) }
func (o *IntTotalOrder) Pred(v int32) int32                { return int32(o.core.pred(int64(v))) }
func (o *IntTotalOrder) Display(out io.Writer) error       { return o.core.display(out, false) }
func (o *IntTotalOrder) DisplayDetail(out io.Writer) error { return o.core.display(out, true) }
func (o *IntTotalOrder) CheckInvariant() bool              { return o.core.checkInvariant() }
func (o *IntTotalOrder) String() string                    { var b strings.Builder; _ = o.Display(&b); return b.String() }
