//go:build g3trace

package params

import (
	"bytes"
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
