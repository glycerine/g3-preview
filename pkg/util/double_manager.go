// Source: src/util/double_manager.h:34-100.
package util

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

// DoubleManager is the source scalar, copyable approximate arithmetic manager.
// In particular IsNeg compares against positive tolerance, exactly as written.
type DoubleManager struct{ zeroTolerance float64 }

func NewDoubleManager(c *ParamsContext, p *ParamsRef) (DoubleManager, error) {
	var m DoubleManager
	e := m.UpdtParams(c, p)
	return m, e
}
func (m *DoubleManager) UpdtParams(c *ParamsContext, p *ParamsRef) error {
	value := 1e-8
	if p != nil {
		if c == nil {
			return ErrParamsOwner
		}
		key := "zero_tolerance"
		var e error
		value, e = c.GetDoubleName(p, &key, value)
		if e != nil {
			return e
		}
	}
	m.zeroTolerance = value
	return nil
}
func (m DoubleManager) ZeroTolerance() float64 { return m.zeroTolerance }
func DoubleManagerPrecise() bool               { return false }
func DoubleManagerReset(a *float64)            { *a = 0 }
func DoubleManagerDel(a *float64)              {} // source deletion erases even a read.
func DoubleManagerAdd(a, b float64) float64    { return a + b }
func DoubleManagerSub(a, b float64) float64    { return a - b }
func DoubleManagerMul(a, b float64) float64    { return a * b }
func DoubleManagerDiv(a, b float64) float64    { return a / b }
func DoubleManagerInv(a float64) float64       { return 1 / a }
func DoubleManagerInvInPlace(a *float64)       { *a = 1 / *a }
func DoubleManagerNeg(a *float64)              { *a = -*a }
func DoubleManagerAbs(a *float64) {
	if *a < 0 {
		DoubleManagerNeg(a)
	}
}
func DoubleManagerAddmul(a, b, c float64) float64 { product := float64(b * c); return a + product }
func DoubleManagerSubmul(a, b, c float64) float64 { product := float64(b * c); return a - product }

var ErrDoubleManagerPower = errors.New("g3: double_manager debug power exceeds INT_MAX")
var ErrDoubleManagerHashDomain = errors.New("g3: double_manager hash conversion outside defined uint64 domain")

func DoubleManagerPower(a float64, p uint32) (float64, error) {
	if sourceDebug && p > math.MaxInt32 {
		return 0, ErrDoubleManagerPower
	}
	// The selected source int is signed32; release narrowing is preserved.
	return math.Pow(a, float64(int32(p))), nil
}
func DoubleManagerFloor(a float64) float64            { return math.Floor(a) }
func DoubleManagerCeil(a float64) float64             { return math.Ceil(a) }
func (m DoubleManager) Eq(a, b float64) bool          { return m.IsZero(a - b) }
func (m DoubleManager) Neq(a, b float64) bool         { return !m.Eq(a, b) }
func DoubleManagerLt(a, b float64) bool               { return a < b }
func DoubleManagerLe(a, b float64) bool               { return a <= b }
func DoubleManagerGt(a, b float64) bool               { return a > b }
func DoubleManagerGe(a, b float64) bool               { return a >= b }
func DoubleManagerSetFraction(a *float64, n, d int32) { *a = float64(n) / float64(d) }
func DoubleManagerSet(a *float64, n float64)          { *a = n }
func DoubleManagerSetInt32(a *float64, n int32)       { *a = float64(n) }
func DoubleManagerSetUint32(a *float64, n uint32)     { *a = float64(n) }
func DoubleManagerSetInt64(a *float64, n int64)       { *a = float64(n) }
func DoubleManagerSetUint64(a *float64, n uint64)     { *a = float64(n) }
func DoubleManagerSwap(a, b *float64)                 { *a, *b = *b, *a }
func (m DoubleManager) IsPos(a float64) bool          { return a > m.zeroTolerance }
func (m DoubleManager) IsNeg(a float64) bool          { return a < m.zeroTolerance }
func (m DoubleManager) IsZero(a float64) bool         { return -m.zeroTolerance <= a && a <= m.zeroTolerance }
func (m DoubleManager) IsNonPos(a float64) bool       { return !m.IsPos(a) }
func (m DoubleManager) IsNonNeg(a float64) bool       { return !m.IsNeg(a) }
func DoubleManagerIsOne(a float64) bool               { return a == 1 }
func DoubleManagerIsMinusOne(a float64) bool          { return a == -1 }
func DoubleManagerIsInt(a float64) bool               { return a == math.Floor(a) }
func DoubleManagerHash(a float64) (uint32, error) {
	n := math.Trunc(a)
	if math.IsNaN(n) || n < 0 || n >= 0x1p64 {
		return 0, ErrDoubleManagerHashDomain
	}
	return Uint64(uint64(n)), nil
}

// SetString uses the selected C-locale atof consumed prefix, including optional
// hex exponent and incomplete exponent rollback. Decimal/hex conversion uses
// Go's IEEE64 parser; C errno/locale/rounding and non-IEEE64 NaN payload ABI are qualified.
func DoubleManagerSetString(a *float64, text string) { *a = doubleManagerAtOf(text) }
func doubleManagerAtOf(text string) float64 {
	if end := strings.IndexByte(text, 0); end >= 0 {
		text = text[:end]
	}
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n' || text[i] == '\r' || text[i] == '\v' || text[i] == '\f') {
		i++
	}
	start := i
	negative := false
	if i < len(text) && (text[i] == '+' || text[i] == '-') {
		negative = text[i] == '-'
		i++
	}
	if len(text)-i >= 3 {
		prefix := text[i : i+3]
		if strings.EqualFold(prefix, "inf") {
			if negative {
				return math.Inf(-1)
			}
			return math.Inf(1)
		}
		if strings.EqualFold(prefix, "nan") {
			v := doubleManagerNaN(text[i+3:])
			if negative {
				v = math.Copysign(v, -1)
			}
			return v
		}
	}
	hex := i+2 <= len(text) && text[i] == '0' && (text[i+1] == 'x' || text[i+1] == 'X')
	if hex {
		i += 2
	}
	count := 0
	for i < len(text) && doubleManagerDigit(text[i], hex) {
		i++
		count++
	}
	if i < len(text) && text[i] == '.' {
		i++
		for i < len(text) && doubleManagerDigit(text[i], hex) {
			i++
			count++
		}
	}
	if count == 0 {
		// An uncompleted 0x prefix still has the decimal leading zero.
		if hex {
			v := 0.0
			if negative {
				v = math.Copysign(v, -1)
			}
			return v
		}
		return 0
	}
	exponent := byte('e')
	if hex {
		exponent = 'p'
	}
	hasExponent := false
	if i < len(text) && (text[i] == exponent || text[i] == exponent-32) {
		mark := i
		i++
		if i < len(text) && (text[i] == '+' || text[i] == '-') {
			i++
		}
		begin := i
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
		}
		if i == begin {
			i = mark
		} else {
			hasExponent = true
		}
	}
	number := text[start:i]
	if hex && !hasExponent {
		number += "p0"
	}
	v, _ := strconv.ParseFloat(number, 64)
	return v
}

// Selected glibc strtod_nan_main.c:24-58: an ASCII n-char sequence
// ending in ')' uses fully consumed base-0 strtoull, with overflow saturation.
// The selected IEEE64 profile retains low payload bits and forces quiet NaN.
func doubleManagerNaN(tail string) float64 {
	const quiet = uint64(0x7ff8000000000000)
	if len(tail) == 0 || tail[0] != '(' {
		return math.Float64frombits(quiet)
	}
	end := 1
	for end < len(tail) && (tail[end] >= '0' && tail[end] <= '9' || tail[end] >= 'A' && tail[end] <= 'Z' || tail[end] >= 'a' && tail[end] <= 'z' || tail[end] == '_') {
		end++
	}
	if end == len(tail) || tail[end] != ')' {
		return math.Float64frombits(quiet)
	}
	token := tail[1:end]
	base := uint64(10)
	start := 0
	if len(token) > 0 && token[0] == '0' {
		base = 8
	}
	if len(token) > 2 && token[0] == '0' && (token[1] == 'x' || token[1] == 'X') {
		base = 16
		start = 2
	}
	var value uint64
	for i := start; i < len(token); i++ {
		ch := token[i]
		var digit uint64
		switch {
		case ch >= '0' && ch <= '9':
			digit = uint64(ch - '0')
		case ch >= 'a' && ch <= 'f':
			digit = uint64(ch - 'a' + 10)
		case ch >= 'A' && ch <= 'F':
			digit = uint64(ch - 'A' + 10)
		default:
			return math.Float64frombits(quiet)
		}
		if digit >= base {
			return math.Float64frombits(quiet)
		}
		if value > (math.MaxUint64-digit)/base {
			value = math.MaxUint64
		} else {
			value = value*base + digit
		}
	}
	return math.Float64frombits(quiet | value&((uint64(1)<<51)-1))
}

func doubleManagerDigit(ch byte, hex bool) bool {
	return ch >= '0' && ch <= '9' || hex && (ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F')
}

// DoubleManagerToString represents the source local ostringstream in fresh
// caller-provided scope storage. On success dst owns str() and the stream is
// destroyed; on failure both actual objects remain for caller OwnAbort. This
// explicit lifetime interface replaces native stack unwinding, not formatting.
func DoubleManagerToString(a float64, b *arena.Budget, local *OwnedStringStream, dst *OwnedString) error {
	if local == nil || dst == nil || dst.Self != nil {
		return ErrParamsPrecondition
	}
	if e := local.Construct(b); e != nil {
		return e
	}
	var digits [48]byte
	var text []byte
	if math.IsNaN(a) {
		if math.Signbit(a) {
			text = []byte("-nan")
		} else {
			text = []byte("nan")
		}
	} else if math.IsInf(a, 1) {
		text = []byte("inf")
	} else if math.IsInf(a, -1) {
		text = []byte("-inf")
	} else {
		text = strconv.AppendFloat(digits[:0], a, 'g', 12, 64)
	}
	for _, ch := range text {
		if e := local.emit(ch); e != nil {
			return e
		}
	}
	if e := local.Str(dst); e != nil {
		return e
	}
	local.OwnAbort()
	return nil
}
