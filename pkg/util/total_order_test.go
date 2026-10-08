// Source: src/test/total_order.cpp:22-144; src/util/total_order.h:28-406.
package util

import (
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestUintTotalOrderRegisteredBasicAssertions(t *testing.T) {
	o := NewUintTotalOrder()
	o.Insert(1)
	o.InsertAfter(1, 2)
	o.InsertAfter(1, 3)
	if !o.LT(1, 2) || !o.LT(3, 2) || !o.LT(1, 3) || o.LT(2, 3) || o.LT(3, 1) || o.LT(2, 2) {
		t.Fatal("source ordering", o.String())
	}
	if o.String() != "1 3 2" || o.First() != 1 || o.Pred(2) != 3 || o.Next(1) != 3 || !o.CheckInvariant() {
		t.Fatal("navigation", o.String())
	}
}

func TestUintTotalOrderRegisteredRepeatedMoves(t *testing.T) {
	o := NewUintTotalOrder()
	o.Insert(1)
	o.InsertAfter(1, 2)
	o.InsertAfter(2, 3)
	for range 1000 {
		o.MoveAfter(3, 1)
		o.MoveAfter(1, 2)
		o.MoveAfter(2, 3)
		if !o.LT(1, 2) || !o.LT(2, 3) || !o.CheckInvariant() {
			t.Fatal("repeated move", o.String())
		}
	}
}

func TestUintTotalOrderChaCha8MovesAgainstSliceOracle(t *testing.T) {
	const size = 100
	o := NewUintTotalOrder()
	oracle := make([]uint32, size)
	for i := range oracle {
		oracle[i] = uint32(i)
		if i == 0 {
			o.Insert(0)
		} else {
			o.InsertAfter(uint32(i-1), uint32(i))
		}
	}
	random := rand.New(rand.NewChaCha8([32]byte{41, 17, 93, 6}))
	for round := 0; round < 20000; round++ {
		a, b := uint32(random.IntN(size)), uint32(random.IntN(size))
		if a != b {
			o.MoveAfter(a, b)
			oracleMoveAfter(oracle, a, b)
		}
		if round%97 == 0 {
			if got := uintTotalOrderValues(o); !reflect.DeepEqual(got, oracle) || !o.CheckInvariant() {
				t.Fatal("oracle mismatch", round, got, oracle)
			}
		}
	}
}

func oracleMoveAfter(values []uint32, after, moved uint32) {
	positionAfter, positionMoved := -1, -1
	for i, value := range values {
		if value == after {
			positionAfter = i
		}
		if value == moved {
			positionMoved = i
		}
	}
	copy(values[positionMoved:], values[positionMoved+1:])
	if positionMoved < positionAfter {
		positionAfter--
	}
	copy(values[positionAfter+2:], values[positionAfter+1:len(values)-1])
	values[positionAfter+1] = moved
}

func uintTotalOrderValues(o *UintTotalOrder) []uint32 {
	result := make([]uint32, 0, o.Size())
	if o.Empty() {
		return result
	}
	for value := o.First(); ; value = o.Next(value) {
		result = append(result, value)
		if !o.HasNext(value) {
			break
		}
	}
	return result
}

func TestTotalOrderMoveBeginningEraseDetailAndIntSpecialization(t *testing.T) {
	o := NewUintTotalOrder()
	for _, value := range []uint32{4, 3, 2, 1} {
		o.Insert(value)
	}
	if o.String() != "1 2 3 4" {
		t.Fatal("front insertion", o.String())
	}
	o.MoveBeginning(3)
	o.Erase(2)
	if o.String() != "3 1 4" || o.Size() != 3 || o.Contains(2) || o.HasPred(3) || o.HasNext(4) || !o.GT(4, 1) {
		t.Fatal("move/erase", o.String())
	}
	var detail strings.Builder
	if err := o.DisplayDetail(&detail); err != nil || !strings.Contains(detail.String(), "3:") || !o.CheckInvariant() {
		t.Fatal("detail", detail.String(), err)
	}

	signed := NewIntTotalOrder()
	signed.Insert(-1)
	signed.InsertAfter(-1, -3)
	signed.InsertAfter(-3, 2)
	if signed.String() != "-1 -3 2" || !signed.LT(-3, 2) || signed.First() != -1 || signed.Next(-1) != -3 || signed.Pred(2) != -3 || !signed.CheckInvariant() {
		t.Fatal("signed specialization", signed.String())
	}
}
