// Source: src/util/resource_limit.h:39-164.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2007-04-18.
package util

import (
	"errors"
	"math"
)

var errSearchResourceLimitDomain = errors.New("g3: undefined signed search resource limit arithmetic")

// The concrete values translate the source base, linear and exponential
// classes. Descriptions borrow immutable Go spelling storage, truncated at its
// first NUL. These are caller-owned cold values, without owned backing or a
// synthetic allocation. The source TRACE stream binding remains separate.
type utilBaseResourceLimit struct {
	counter, current, initial, maximum int32
	description                        string
	descriptionPresent                 bool
}

// Cold polymorphic contract corresponding to the source abstract class.
type utilSearchResourceLimit interface {
	getDescription() (string, bool)
	resetCounters()
	exhausted() bool
	incLimit() (bool, error)
	getLimit() int32
	reset()
}

type utilLinearResourceLimit struct {
	utilBaseResourceLimit
	increment int32
}

type utilExponentialResourceLimit struct {
	utilBaseResourceLimit
	ratio float64
}

func newUtilBaseResourceLimit(initial, maximum int32, description *string) utilBaseResourceLimit {
	r := utilBaseResourceLimit{current: initial, initial: initial, maximum: maximum}
	if description != nil {
		r.description, r.descriptionPresent = (*description)[:cSymbolLength(*description)], true
	}
	return r
}

func newUtilLinearResourceLimit(initial, maximum, increment int32, description *string) utilLinearResourceLimit {
	return utilLinearResourceLimit{newUtilBaseResourceLimit(initial, maximum, description), increment}
}

func newUtilExponentialResourceLimit(initial, maximum int32, ratio float64, description *string) utilExponentialResourceLimit {
	return utilExponentialResourceLimit{newUtilBaseResourceLimit(initial, maximum, description), ratio}
}

func (r *utilBaseResourceLimit) getCounter() int32 { return r.counter }
func (r *utilBaseResourceLimit) getDescription() (string, bool) {
	return r.description, r.descriptionPresent
}
func (r *utilBaseResourceLimit) getLimit() int32      { return r.current }
func (r *utilBaseResourceLimit) exhausted() bool      { return r.counter >= r.current }
func (r *utilBaseResourceLimit) resetCounters()       { r.counter = 0 }
func (r *utilBaseResourceLimit) setMaxLimit(n int32)  { r.maximum = n }
func (r *utilBaseResourceLimit) setInitLimit(n int32) { r.initial, r.current = n, n }
func (r *utilBaseResourceLimit) reset()               { r.counter, r.current = 0, r.initial }

func (r *utilBaseResourceLimit) incCounter() (bool, error) {
	// Signed C++ overflow is outside the defined source domain. Guard it at
	// the reached increment, without publishing a wrapped count.
	if r.counter == math.MaxInt32 {
		return false, errSearchResourceLimitDomain
	}
	r.counter++
	return r.counter <= r.current, nil
}

func (r *utilLinearResourceLimit) incLimit() (bool, error) {
	n := int64(r.current) + int64(r.increment)
	if n < math.MinInt32 || n > math.MaxInt32 {
		return false, errSearchResourceLimitDomain
	}
	if r.maximum < 0 || int32(n) <= r.maximum {
		r.current = int32(n)
		return true, nil
	}
	return false, nil
}

func (r *utilExponentialResourceLimit) incLimit() (bool, error) {
	// Source multiplies in double and converts to int BEFORE the max test.
	// Guard nonfinite or out-of-range conversions at that actual operation.
	x := math.Trunc(float64(r.current) * r.ratio)
	if math.IsNaN(x) || x < math.MinInt32 || x > math.MaxInt32 {
		return false, errSearchResourceLimitDomain
	}
	n := int32(x)
	if r.maximum < 0 || n <= r.maximum {
		r.current = n
		return true, nil
	}
	return false, nil
}
