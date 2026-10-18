package polynomial

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: factorization.cpp:238-245. Each actual normalized numeral is
// followed by a tab, including the last column; each complete row has endl.
func (q *upolyBerlekampMatrix) display(out io.Writer) error {
	for i := uint32(0); i < q.values.size/q.size; i++ {
		for j := uint32(0); j < q.size; j++ {
			value, e := q.getConst(i, j)
			if e != nil {
				return e
			}
			text, e := q.m.modular.String(value)
			if e != nil {
				return e
			}
			if e = linearWriteText(out, text); e != nil {
				return e
			}
			if e = linearWriteText(out, "\t"); e != nil {
				return e
			}
		}
		if e := diagnostic.WriteTraceEndl(out); e != nil {
			return e
		}
	}
	return nil
}

// Complete source records at142 and198. The borrow spans header/body/footer,
// including source row flushes, while constructor/diagonalization NUMs live.
func (q *upolyBerlekampMatrix) traceMatrix(line int) error {
	if q.m != nil && q.m.pendingBerlekamp != nil {
		return ErrUPolynomialCleanupPending
	}
	tags := q.m.modular.BorrowTraceTags()
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		return nil
	}
	if q.displaying {
		return ErrUPolynomialCleanupPending
	}
	q.displaying = true
	defer func() { q.displaying = false }()
	if e := q.traceBorrow.bindManagers([3]*UPolynomialManager{q.m}, tags); e != nil {
		return e
	}
	defer q.traceBorrow.unbind()
	name, caption := "berlekamp_matrix", "polynomial::berlekamp_matrix():"
	if line == 198 {
		name, caption = "diagonalize", "polynomial::diagonalize():"
	}
	var computation error
	_ = tags.EmitConditional("polynomial::factorization::bughunt", name, "src/math/polynomial/upolynomial_factorization.cpp", line,
		func() bool { return true }, func(raw io.Writer) error {
			out := factorizationTraceWriter{raw}
			_ = linearWriteText(out, caption)
			_ = diagnostic.WriteTraceEndl(out)
			if e := q.display(out); e != nil {
				computation = e
				return e
			}
			return diagnostic.WriteTraceEndl(out)
		})
	return computation
}

// Source: factorization.cpp:97. Pivot members already exist, but neither
// the first matrix coefficient nor the constructor's scoped tmp exists yet.
func (q *upolyBerlekampMatrix) traceConstructor(f UPolynomialView, p uint32) error {
	tags := q.m.modular.BorrowTraceTags()
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		return nil
	}
	if q.displaying {
		return ErrUPolynomialCleanupPending
	}
	q.displaying = true
	defer func() { q.displaying = false }()
	if e := q.traceBorrow.bindManagers([3]*UPolynomialManager{q.m}, tags); e != nil {
		return e
	}
	defer q.traceBorrow.unbind()
	var computation error
	_ = tags.EmitConditional("polynomial::factorization::bughunt", "berlekamp_matrix", "src/math/polynomial/upolynomial_factorization.cpp", 97,
		func() bool { return true }, func(raw io.Writer) error {
			out := factorizationTraceWriter{raw}
			_ = linearWriteText(out, "polynomial::berlekamp_matrix(")
			var issued upolyDisplayInvocationReceipt
			e := q.m.displayIntoWithReceipt(f, factorizationTraceStream{out}, "x", false, &issued)
			if q.m.matchesDisplayReceipt(&issued) {
				q.entryTrace = issued
			}
			if e != nil {
				computation = e
				return e
			}
			_ = linearWriteText(out, ", ")
			_ = linearWriteText(out, strconv.FormatUint(uint64(p), 10))
			_ = linearWriteText(out, ")")
			return diagnostic.WriteTraceEndl(out)
		})
	return computation
}
