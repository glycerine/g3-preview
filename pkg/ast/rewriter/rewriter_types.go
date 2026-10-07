// Source: src/ast/rewriter/rewriter_types.h:29-49.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package rewriter

type brStatus uint32

const (
	brRewrite1 brStatus = iota
	brRewrite2
	brRewrite3
	brRewriteFull
	brDone
	brFailed
)

func unsignedToBRStatus(u uint32) brStatus {
	if u >= 3 {
		return brRewriteFull
	}
	return brStatus(u)
}
