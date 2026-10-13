// Source: src/util/mpzzp.h:55,175,177,179.
package numeral

import (
	"fmt"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *IntegerModularManager) BindTrace(tags *diagnostic.TraceTags) error {
	if e := m.check(); e != nil {
		return e
	}
	m.trace = tags
	return nil
}

func (m *IntegerModularManager) modularTraceTags() *diagnostic.TraceTags {
	if m.trace != nil {
		return m.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (m *IntegerModularManager) traceSetupP() error {
	tags := m.modularTraceTags()
	if !tags.ShouldTrace("mpzzp") {
		return nil
	}
	lower, e := m.manager.String(&m.Lower)
	if e != nil {
		return e
	}
	upper, e := m.manager.String(&m.Upper)
	if e != nil {
		return e
	}
	_ = tags.Emit("mpzzp", "setup_p", "src/util/mpzzp.h", 55,
		fmt.Sprintf("lower: %s, upper: %s\n", lower, upper))
	return nil
}

func (m *IntegerModularManager) traceInvStart(a *Int) error {
	tags := m.modularTraceTags()
	if !tags.ShouldTrace("mpzp_inv_bug") {
		return nil
	}
	value, e := m.manager.String(a)
	if e != nil {
		return e
	}
	prime, e := m.manager.String(&m.P)
	if e != nil {
		return e
	}
	_ = tags.Emit("mpzp_inv_bug", "inv", "src/util/mpzzp.h", 175,
		fmt.Sprintf("a: %s, p: %s\n", value, prime))
	return nil
}

func (m *IntegerModularManager) traceInvNormalized(a *Int) error {
	tags := m.modularTraceTags()
	if !tags.ShouldTrace("mpzp_inv_bug") {
		return nil
	}
	value, e := m.manager.String(a)
	if e != nil {
		return e
	}
	_ = tags.Emit("mpzp_inv_bug", "inv", "src/util/mpzzp.h", 177,
		"after normalization a: "+value+"\n")
	return nil
}

func (m *IntegerModularManager) traceInvGCD() error {
	tags := m.modularTraceTags()
	if !tags.ShouldTrace("mpzp_inv_bug") {
		return nil
	}
	first, e := m.manager.String(&m.InvTmp1)
	if e != nil {
		return e
	}
	second, e := m.manager.String(&m.InvTmp2)
	if e != nil {
		return e
	}
	third, e := m.manager.String(&m.InvTmp3)
	if e != nil {
		return e
	}
	_ = tags.Emit("mpzp_inv_bug", "inv", "src/util/mpzzp.h", 179,
		fmt.Sprintf("tmp1: %s\ntmp2: %s\ntmp3: %s\n", first, second, third))
	return nil
}
