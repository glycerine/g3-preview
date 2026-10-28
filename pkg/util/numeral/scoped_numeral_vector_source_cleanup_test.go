package numeral

// Source: src/util/scoped_numeral_vector.h:29-58, concrete mpz/mpq vectors.

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedIntVectorStoppedSourceKeepsAscendingSuffix(t *testing.T) {
	for _, route := range []string{"shrink", "reset", "close"} {
		t.Run(route, func(t *testing.T) {
			q, _, _ := rawArithmeticGuardOwner(t)
			var source Int
			serviceMust(t, q.Z.SetLimbs(&source, []uint32{7, 0, 1}))
			t.Cleanup(func() { serviceMust(t, q.Z.Del(&source)) })
			baseline := q.Z.Cells.Stats().Live
			v, e := NewScopedIntVector(q.Z)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, v.Cleanup()) })
			for range 3 {
				serviceMust(t, v.Push(&source))
			}
			first, middle, last := v.data[0].Cell, v.data[1].Cell, v.data[2].Cell
			original := q.Z.Cell(middle)
			bad := original
			bad.Backing.Requested = uint64(math.MaxUint32) + 1
			q.Z.Store(middle, bad)
			corrupted := true
			t.Cleanup(func() {
				if corrupted {
					q.Z.Store(middle, original)
				}
			})
			var stop error
			switch route {
			case "shrink":
				stop = v.Shrink(0)
			case "reset":
				stop = v.Reset()
			default:
				stop = v.Close()
			}
			if !errors.Is(stop, arena.ErrBounds) || v.closed || v.Size() != 3 || v.data[0].Cell != 0 || v.data[1].Cell != middle || v.data[2].Cell != last || first == middle {
				t.Fatal("source loop consumed stopped integer suffix", stop, v.data)
			}
			if again := v.Close(); again != stop || v.data[2].Cell != last {
				t.Fatal("stopped vector source destructor replayed", again)
			}
			if e := v.Cleanup(); !errors.Is(e, arena.ErrBounds) || v.closed || v.data[2].Cell != last {
				t.Fatal("owning refusal lost integer suffix", e)
			}
			q.Z.Store(middle, original)
			corrupted = false
			serviceMust(t, v.Cleanup())
			if !v.closed || q.Z.Cells.Stats().Live != baseline {
				t.Fatal("owning cleanup retained integer vector", q.Z.Cells.Stats().Live, baseline)
			}
		})
	}
}

func TestScopedRationalVectorStoppedSourceKeepsComponentSuffix(t *testing.T) {
	for _, route := range []string{"shrink", "reset", "close"} {
		t.Run(route, func(t *testing.T) {
			q, _, _ := rawArithmeticGuardOwner(t)
			source := NewRational()
			serviceMust(t, q.Z.SetLimbs(&source.Num, []uint32{7, 0, 1}))
			serviceMust(t, q.Z.SetLimbs(&source.Den, []uint32{9, 0, 1}))
			t.Cleanup(func() { serviceMust(t, q.Del(&source)) })
			baseline := q.Z.Cells.Stats().Live
			v, e := NewScopedRationalVector(q)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, v.Cleanup()) })
			for range 3 {
				serviceMust(t, v.Push(&source))
			}
			middleDen, lastNum, lastDen := v.data[1].Den.Cell, v.data[2].Num.Cell, v.data[2].Den.Cell
			original := q.Z.Cell(middleDen)
			bad := original
			bad.Backing.Requested = uint64(math.MaxUint32) + 1
			q.Z.Store(middleDen, bad)
			corrupted := true
			t.Cleanup(func() {
				if corrupted {
					q.Z.Store(middleDen, original)
				}
			})
			var stop error
			switch route {
			case "shrink":
				stop = v.Shrink(0)
			case "reset":
				stop = v.Reset()
			default:
				stop = v.Close()
			}
			if !errors.Is(stop, arena.ErrBounds) || v.closed || v.Size() != 3 || v.data[0].Num.Cell != 0 || v.data[0].Den.Cell != 0 || v.data[1].Num.Cell != 0 || v.data[1].Den.Cell != middleDen || v.data[2].Num.Cell != lastNum || v.data[2].Den.Cell != lastDen {
				t.Fatal("source loop consumed stopped rational suffix", stop, v.data)
			}
			if again := v.Close(); again != stop || v.data[2].Den.Cell != lastDen {
				t.Fatal("stopped rational vector destructor replayed", again)
			}
			if e := v.Cleanup(); !errors.Is(e, arena.ErrBounds) || v.closed || v.data[2].Den.Cell != lastDen {
				t.Fatal("owning refusal lost rational suffix", e)
			}
			q.Z.Store(middleDen, original)
			corrupted = false
			serviceMust(t, v.Cleanup())
			if !v.closed || q.Z.Cells.Stats().Live != baseline {
				t.Fatal("owning cleanup retained rational vector", q.Z.Cells.Stats().Live, baseline)
			}
		})
	}
}
