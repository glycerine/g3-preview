package params

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestGlobalParamsNamespaceDirectWriterAndStickyRefusal(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	paramsMust(t, s.InitializeSource())
	var want bytes.Buffer
	paramsMust(t, s.SourceDisplayInto(&want, 0, false, false))
	var direct bytes.Buffer
	paramsMust(t, s.SourceDisplayTo(&direct, 0, false, false))
	if direct.String() != want.String() {
		t.Fatal("direct source stream differs from byte-buffer display")
	}
	for _, short := range []bool{false, true} {
		for _, limit := range []int{0, 1, 7, 18, 35, len(want.String()) - 1} {
			w := &limitedDescriptorWriter{limit: limit, short: short}
			err := s.SourceDisplayTo(w, 0, false, false)
			kind := error(io.ErrClosedPipe)
			if short {
				kind = io.ErrShortWrite
			}
			if !errors.Is(err, kind) || w.out.String() != want.String()[:limit] || w.calls == 0 {
				t.Fatal("direct global stream failbit prefix", short, limit, err, w.out.String(), w.calls)
			}
		}
	}
	var modules, directModules bytes.Buffer
	paramsMust(t, s.SourceDisplayModulesInto(&modules))
	paramsMust(t, s.SourceDisplayModulesTo(&directModules))
	if modules.String() != directModules.String() {
		t.Fatal("module list direct stream")
	}
	var parameter, diagnostic, directParameter bytes.Buffer
	paramsMust(t, s.SourceDisplayParameterInto(&parameter, &diagnostic, "verbose"))
	paramsMust(t, s.SourceDisplayParameterTo(&directParameter, "verbose"))
	if directParameter.String() != diagnostic.String()+parameter.String() {
		t.Fatal("parameter diagnostic/output source order", directParameter.String())
	}
}

func TestGlobalParamsDirectWriterFailbitStillRegistersCatalog(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	paramsMust(t, s.InitializeSource())
	g, err := s.Manager()
	paramsMust(t, err)
	if g.modulesRegistered {
		t.Fatal("catalog already registered before display")
	}
	w := &limitedDescriptorWriter{limit: 0}
	if err := s.SourceDisplayTo(w, 0, false, false); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("source stream refusal", err)
	}
	if !g.modulesRegistered || w.calls != 1 || w.out.Len() != 0 {
		t.Fatal("source display stopped before catalog registration", g.modulesRegistered, w.calls)
	}
}
