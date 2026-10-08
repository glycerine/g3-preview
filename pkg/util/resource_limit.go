// Source: src/util/resource_limit.h:39-164.
package util

import (
	"errors"
	"math"
)

var ErrResourceLimitDomain = errors.New("g3: undefined signed search resource limit arithmetic")

// The concrete values translate the source base, linear and exponential
// classes. Descriptions borrow immutable Go spelling storage, truncated at its
// first NUL. These are caller-owned cold values, without owned backing or a
// synthetic allocation. The source TRACE stream binding remains separate.
type BaseResourceLimit struct {
	counter, current, initial, maximum int32
	description                        string
	descriptionPresent                 bool
}

// Cold polymorphic contract corresponding to the source abstract class.
type ResourceLimit interface {
	GetDescription() (string, bool)
	ResetCounters()
	Exhausted() bool
	IncLimit() (bool, error)
	GetLimit() int32
	Reset()
}

type LinearResourceLimit struct {
	BaseResourceLimit
	increment int32
}

type ExponentialResourceLimit struct {
	BaseResourceLimit
	ratio float64
}

func newBaseResourceLimit(initial, maximum int32, description *string) BaseResourceLimit {
	r := BaseResourceLimit{current: initial, initial: initial, maximum: maximum}
	if description != nil {
		r.description, r.descriptionPresent = (*description)[:cSymbolLength(*description)], true
	}
	return r
}

func NewLinearResourceLimit(initial, maximum, increment int32, description *string) LinearResourceLimit {
	return LinearResourceLimit{newBaseResourceLimit(initial, maximum, description), increment}
}

func NewExponentialResourceLimit(initial, maximum int32, ratio float64, description *string) ExponentialResourceLimit {
	return ExponentialResourceLimit{newBaseResourceLimit(initial, maximum, description), ratio}
}

func (r *BaseResourceLimit) GetCounter() int32 { return r.counter }
func (r *BaseResourceLimit) GetDescription() (string, bool) {
	return r.description, r.descriptionPresent
}
func (r *BaseResourceLimit) GetLimit() int32      { return r.current }
func (r *BaseResourceLimit) Exhausted() bool      { return r.counter >= r.current }
func (r *BaseResourceLimit) ResetCounters()       { r.counter = 0 }
func (r *BaseResourceLimit) SetMaxLimit(n int32)  { r.maximum = n }
func (r *BaseResourceLimit) SetInitLimit(n int32) { r.initial, r.current = n, n }
func (r *BaseResourceLimit) Reset()               { r.counter, r.current = 0, r.initial }

func (r *BaseResourceLimit) IncCounter() (bool, error) {
	// Signed C++ overflow is outside the defined source domain. Guard it at
	// the reached increment, without publishing a wrapped count.
	if r.counter == math.MaxInt32 {
		return false, ErrResourceLimitDomain
	}
	r.counter++
	return r.counter <= r.current, nil
}

func (r *LinearResourceLimit) IncLimit() (bool, error) {
	n := int64(r.current) + int64(r.increment)
	if n < math.MinInt32 || n > math.MaxInt32 {
		return false, ErrResourceLimitDomain
	}
	if r.maximum < 0 || int32(n) <= r.maximum {
		r.current = int32(n)
		return true, nil
	}
	return false, nil
}

func (r *ExponentialResourceLimit) IncLimit() (bool, error) {
	// Source multiplies in double and converts to int BEFORE the max test.
	// Guard nonfinite or out-of-range conversions at that actual operation.
	x := math.Trunc(float64(r.current) * r.ratio)
	if math.IsNaN(x) || x < math.MinInt32 || x > math.MaxInt32 {
		return false, ErrResourceLimitDomain
	}
	n := int32(x)
	if r.maximum < 0 || n <= r.maximum {
		r.current = n
		return true, nil
	}
	return false, nil
}
