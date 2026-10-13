//go:build g3trace

package params

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

type gparamsTraceBodyRefusalWriter struct {
	bytes.Buffer
	calls, failAt int
	cause         error
}

func (w *gparamsTraceBodyRefusalWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, w.cause
	}
	return w.Buffer.Write(p)
}
func (w *gparamsTraceBodyRefusalWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestGlobalParamsSetNameTraceBodyInsertionPrefixes(t *testing.T) {
	const header = "-------- [gparams] set src/util/gparams.cpp:563 ---------\n"
	parts := [...]string{"setting [", "alpha", "] <- '", "beta", "'\n"}
	for refused := range parts {
		s, _ := gparamsNamespaceFixture(t)
		cause := errors.New("trace body refused")
		out := &gparamsTraceBodyRefusalWriter{failAt: 10 + refused, cause: cause}
		tags := udiag.NewTraceTagsWithWriter(out)
		tags.Enable("gparams")
		paramsMust(t, s.BindNamespaceTrace(tags))
		if err := s.SourceSetName("alpha\x00ignored", "beta\x00ignored"); !errors.Is(err, cause) {
			t.Fatal("refused source body insertion", refused, err)
		}
		want := header
		for _, part := range parts[:refused] {
			want += part
		}
		if got := out.String(); got != want || out.calls != 10+refused {
			t.Fatal("source trace insertion prefix", refused, got, want, out.calls)
		}
	}
}

func TestGlobalParamsSetNameTracePrecedesNullAssertion(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	tags.Enable("gparams")
	paramsMust(t, s.BindNamespaceTrace(tags))
	if e := s.SourceSetName("alpha\x00ignored", "beta\x00ignored"); !namespaceNullError(e) {
		t.Fatal("source null assertion after trace", e)
	}
	want := "-------- [gparams] set src/util/gparams.cpp:563 ---------\nsetting [alpha] <- 'beta'\n------------------------------------------------\n"
	if output.String() != want {
		t.Fatal("source pre-assertion TRACE", output.String())
	}
}

func TestGlobalParamsInitGetRefFinalizeTraceOrder(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	tags.Enable("gparams")
	paramsMust(t, s.BindNamespaceTrace(tags))
	if _, e := s.SourceGetRef(); !namespaceNullError(e) {
		t.Fatal("get_ref trace must precede null assertion", e)
	}
	paramsMust(t, s.InitializeSource())
	if _, e := s.SourceGetRef(); e != nil {
		t.Fatal(e)
	}
	paramsMust(t, s.FinalizeSource())
	if _, e := s.SourceGetRef(); !namespaceNullError(e) {
		t.Fatal("finalized get_ref trace must precede null assertion", e)
	}
	record := func(function string, line int, body string) string {
		return fmt.Sprintf("-------- [gparams] %s src/util/gparams.cpp:%d ---------\n%s------------------------------------------------\n", function, line, body)
	}
	want := record("get_ref", 608, "gparams::get_ref()\n") +
		record("init", 634, "gparams::init()\n") +
		record("get_ref", 608, "gparams::get_ref()\n") +
		record("finalize", 639, "gparams::finalize()\n") +
		record("get_ref", 608, "gparams::get_ref()\n")
	if output.String() != want {
		t.Fatal("source trace order", output.String())
	}
}
