package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func enum2BVParameters(t *testing.T, m *ast.ASTManager) (*uparams.ParamsContext, *uparams.ParamsRef) {
	t.Helper()
	c, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	p, err := c.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() {
		if p != (uparams.ParamsRef{}) {
			astTestMust(t, c.Del(&p))
		}
		astTestMust(t, c.Close())
	})
	return c, &p
}

func enum2BVNewForTest(t *testing.T, m *ast.ASTManager) (*Enum2BVRewriter, error) {
	t.Helper()
	c, p := enum2BVParameters(t, m)
	return NewEnum2BVRewriter(m, c, p)
}

func TestEnum2BVParametersSharedCopyAndCleanupRetention(t *testing.T) {
	m, _, _ := enum2BVFixture(t)
	c, p := enum2BVParameters(t, m)
	key := astTestSymbol(t, m, "enum-retained-value")
	astTestMust(t, c.SetUInt(p, key, 17))
	r, err := NewEnum2BVRewriter(m, c, p)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	if c.Stats().References != 2 {
		t.Fatal("constructor did not acquire source params_ref copy", c.Stats())
	}
	if r.parameters != *p {
		t.Fatal("constructor did not share backing")
	}
	// The shared backing is copied on the caller's subsequent mutation.
	astTestMust(t, c.SetUInt(p, key, 29))
	value, err := c.GetUInt(&r.parameters, key, 0)
	astTestMust(t, err)
	if value != 17 {
		t.Fatal("caller mutation changed retained snapshot")
	}
	astTestMust(t, c.Del(p))
	if c.Stats().References != 1 {
		t.Fatal("caller release destroyed retained parameter owner")
	}
	other, err := c.NewValue()
	astTestMust(t, err)
	astTestMust(t, c.SetUInt(&other, key, 101))
	astTestMust(t, r.UpdateParams(&other))
	astTestMust(t, c.Del(&other))
	// Failed replacement must keep the public cleanup's local copy alive after
	// the old imp, its parameter member and its header have been destroyed.
	m.Budget().FailAfter(0)
	err = r.Cleanup()
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.header != 0 || r.parameters != (uparams.ParamsRef{}) || c.Stats().References != 1 {
		t.Fatal("cleanup lost local parameter owner at replacement refusal", err, c.Stats())
	}
	value, err = c.GetUInt(&r.cleanupParameters, key, 0)
	astTestMust(t, err)
	if value != 17 {
		t.Fatal("UpdateParams changed source's retained constructor parameters", value)
	}
	astTestMust(t, r.Cleanup())
	if r.cleanupParameters != (uparams.ParamsRef{}) || c.Stats().References != 1 {
		t.Fatal("cleanup temporary was not retired", c.Stats())
	}
	value, err = c.GetUInt(&r.parameters, key, 0)
	astTestMust(t, err)
	if value != 17 {
		t.Fatal("replacement lost constructor parameters")
	}
	astTestMust(t, r.Close())
	if c.Stats().References != 0 || c.Stats().LiveBlocks != 0 {
		t.Fatal("Close retained params member", c.Stats())
	}
}

func TestEnum2BVParametersReplacementFailurePrefixes(t *testing.T) {
	m, _, _ := enum2BVFixture(t)
	c, p := enum2BVParameters(t, m)
	key := astTestSymbol(t, m, "enum-replacement-param")
	astTestMust(t, c.SetUInt(p, key, 41))
	complete := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		r, err := NewEnum2BVRewriter(m, c, p)
		astTestMust(t, err)
		m.Budget().FailAfter(ordinal)
		err = r.Cleanup()
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected cleanup failure", ordinal, err)
		}
		if err == nil {
			complete = true
			value, e := c.GetUInt(&r.parameters, key, 0)
			astTestMust(t, e)
			if value != 41 || c.Stats().References != 2 {
				t.Fatal("successful replacement snapshot", value, c.Stats())
			}
		} else {
			value, e := c.GetUInt(&r.cleanupParameters, key, 0)
			astTestMust(t, e)
			if value != 41 {
				t.Fatal("failed replacement lost source temporary", ordinal, value)
			}
		}
		astTestMust(t, r.Close())
		if c.Stats().References != 1 {
			t.Fatal("replacement failure retained params owner", ordinal, c.Stats())
		}
		if complete {
			break
		}
	}
	if !complete {
		t.Fatal("replacement allocation census did not finish")
	}
}
