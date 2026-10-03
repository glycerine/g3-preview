// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Marked storage adaptation of util/region.h/.cpp; domain trail rules separate.
package arena

import (
	"errors"
	"reflect"
)

var (
	ErrMark      = errors.New("arena: invalid, foreign, consumed or non-top region mark")
	ErrReentrant = errors.New("arena: mutation during region cleanup")
)

type RegionRef struct {
	Slot       Slot
	Generation Generation
}
type regionIdentity struct{ nonzero byte }

// RegionMark is a cold opaque owner-bound token. Only scalar mark metadata is
// stored in the region. It cannot be used in pointer-free hot record pages.
type RegionMark struct {
	owner  *regionIdentity
	serial uint64
}
type regionMarkState struct {
	serial uint64
	length uint32
}

// Cleanup is a transient caller dispatcher, never stored per record or mark.
// It must not panic or mutate the region. Reads are allowed; all rollback
// records remain live through every cleanup callback. Nil means caller cleanup
// already ran. Callers are responsible for source reference releases and undo.
type Cleanup32 func(RegionRef, uint32)
type Cleanup64 func(RegionRef, uint64)
type RegionStats struct {
	Storage             Stats
	Live, Scopes        uint32
	LogBytes, MarkBytes uint64
}

// reserveRegionLog reserves old+new directory capacity before copying; on a
// predictable failure it leaves the slice and budget untouched. Cold region
// owner headers, caller scratch and cleanup reserves are budgeted by the owner.
func reserveRegionRefs(data []RegionRef, budget *Budget) ([]RegionRef, error) {
	if len(data) < cap(data) {
		return data, nil
	}
	capacity := cap(data)
	if capacity == 0 {
		capacity = 1
	} else {
		if capacity > int(^uint(0)>>1)/2 {
			return data, ErrSlotLimit
		}
		capacity *= 2
	}
	elementBytes := uint64(reflect.TypeOf(RegionRef{}).Size())
	if uint64(capacity) > uint64(^uint(0)>>1)/elementBytes {
		return data, ErrSlotLimit
	}
	amount := uint64(capacity) * elementBytes
	if e := budget.Reserve(amount); e != nil {
		return data, e
	}
	replacement := make([]RegionRef, len(data), capacity)
	copy(replacement, data)
	budget.Release(uint64(cap(data)) * elementBytes)
	return replacement, nil
}

// reserveRegionLog reserves old+new directory capacity before copying; on a
// predictable failure it leaves the slice and budget untouched. Cold region
// owner headers, caller scratch and cleanup reserves are budgeted by the owner.
func reserveRegionMarks(data []regionMarkState, budget *Budget) ([]regionMarkState, error) {
	if len(data) < cap(data) {
		return data, nil
	}
	capacity := cap(data)
	if capacity == 0 {
		capacity = 1
	} else {
		if capacity > int(^uint(0)>>1)/2 {
			return data, ErrSlotLimit
		}
		capacity *= 2
	}
	elementBytes := uint64(reflect.TypeOf(regionMarkState{}).Size())
	if uint64(capacity) > uint64(^uint(0)>>1)/elementBytes {
		return data, ErrSlotLimit
	}
	amount := uint64(capacity) * elementBytes
	if e := budget.Reserve(amount); e != nil {
		return data, e
	}
	replacement := make([]regionMarkState, len(data), capacity)
	copy(replacement, data)
	budget.Release(uint64(cap(data)) * elementBytes)
	return replacement, nil
}
