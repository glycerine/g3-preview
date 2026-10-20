package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestRewriterTypesSourceStatusesAndExceptionHierarchy(t *testing.T) {
	want := []brStatus{brRewrite1, brRewrite2, brRewrite3, brRewriteFull, brRewriteFull}
	for u, status := range want {
		if got := unsignedToBRStatus(uint32(u)); got != status {
			t.Fatalf("unsigned2br_status(%d) = %d, want %d", u, got, status)
		}
	}

	var err error = newRewriterException("rewrite stopped\x00ignored")
	if err.Error() != "rewrite stopped" {
		t.Fatalf("rewriter_exception message = %q", err.Error())
	}
	var concrete rewriterException
	if !errors.As(err, &concrete) {
		t.Fatal("rewriter_exception concrete catch failed")
	}
	var base z3exception.DefaultException
	if !errors.As(err, &base) || base.Message() != "rewrite stopped" {
		t.Fatalf("default_exception base catch = %#v", base)
	}
}
