// Source: src/util/approx_nat.h:26-43; src/util/approx_nat.cpp:22-61.
package util

const approxNatLimit = ^uint32(0) >> 2

// ApproxNat represents a natural number up to UINT_MAX/4, or the huge sentinel.
// Its zero value is the source default constructor. Huge is absorbing on the
// left of arithmetic, including multiplication by zero.
type ApproxNat struct{ value uint32 }

func NewApproxNat(value uint32) ApproxNat {
	var n ApproxNat
	n.Assign(value)
	return n
}

func (n ApproxNat) IsHuge() bool     { return n.value == ^uint32(0) }
func (n ApproxNat) GetValue() uint32 { return n.value }

func (n *ApproxNat) Assign(value uint32) *ApproxNat {
	if value > approxNatLimit {
		n.value = ^uint32(0)
	} else {
		n.value = value
	}
	return n
}

func (n *ApproxNat) AddUnsignedAssign(value uint32) *ApproxNat {
	if n.IsHuge() {
		return n
	}
	if value > approxNatLimit {
		n.value = ^uint32(0)
		return n
	}
	n.value += value
	if n.value > approxNatLimit {
		n.value = ^uint32(0)
	}
	return n
}

func (n *ApproxNat) AddAssign(other ApproxNat) *ApproxNat {
	return n.AddUnsignedAssign(other.value)
}

func (n *ApproxNat) MulUnsignedAssign(value uint32) *ApproxNat {
	if n.IsHuge() {
		return n
	}
	result := uint64(n.value) * uint64(value)
	if result > uint64(approxNatLimit) {
		n.value = ^uint32(0)
	} else {
		n.value = uint32(result)
	}
	return n
}

func (n *ApproxNat) MulAssign(other ApproxNat) *ApproxNat {
	return n.MulUnsignedAssign(other.value)
}

func (n ApproxNat) LessUnsigned(value uint32) bool {
	return !n.IsHuge() && n.value < value
}

// Less preserves the source guard on both operands: finite is not less than huge.
func (n ApproxNat) Less(other ApproxNat) bool {
	return !n.IsHuge() && !other.IsHuge() && n.value < other.value
}

func (n ApproxNat) Display(out *NumericTextStream) error {
	if n.IsHuge() {
		return out.AppendCString("[huge]")
	}
	return out.AppendUint32(n.value)
}
