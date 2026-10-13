//go:build g3trace

package params

import (
	"bytes"
	"fmt"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

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
