package smtcore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestSMTKernelPreprocessorConstructionAllocationFrontiers(t *testing.T) {
	success := false
	for stop := int64(0); stop < 8192 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m, services := smtModelGeneratorFixture(t)
			kernel, err := NewSMTKernel(m, nil)
			astTestMust(t, err)
			p := services.Context.EmptyValue()
			m.Budget().FailAfter(stop)
			err = kernel.configurePreprocessing(services.Context, services.Globals, &p, nil)
			m.Budget().FailAfter(-1)
			astTestMust(t, kernel.Close())
			if err == nil {
				success = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) {
				var memout umemory.Failure
				if !errors.As(err, &memout) || memout.Kind != umemory.FailureMemout {
					t.Fatal(err)
				}
			}
		})
		if t.Failed() {
			t.Fatal("SMT preprocessor frontier", stop)
		}
	}
	if !success {
		t.Fatal("no successful SMT preprocessor frontier")
	}
}
