package protomodel

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestProtoModelConstructorFailureRetainsAllocatedOwner(t *testing.T) {
	success := false
	for stop := int64(0); stop < 512 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m, _, config := protoFixtureWithConfig(t)
			m.Budget().FailAfter(stop)
			fresh, err := NewProtoModel(m, config)
			m.Budget().FailAfter(-1)
			if err == nil {
				success = true
				protoMust(t, fresh.Close())
			} else {
				if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
				if errors.Is(err, ast.ErrASTHandle) {
					t.Fatal("deferred cleanup lost the allocated owner", err)
				}
			}
		})
		if t.Failed() {
			t.Fatal("constructor reservation frontier", stop)
		}
	}
	if !success {
		t.Fatal("no successful constructor frontier")
	}
}
