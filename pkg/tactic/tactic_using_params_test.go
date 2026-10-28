package tactic

// Source: src/tactic/tactical.cpp:1158-1188.

import (
	"testing"

	uparams "github.com/glycerine/g3/pkg/util/params"
)

type paramsRecordingTactic struct {
	*primitiveTactic
	context        *uparams.ParamsContext
	fixed, dynamic string
	seen           [][2]uint32
}

func (t *paramsRecordingTactic) UpdateParams(p *uparams.ParamsRef) error {
	fixed, err := t.context.GetUIntName(p, &t.fixed, 0)
	if err != nil {
		return err
	}
	dynamic, err := t.context.GetUIntName(p, &t.dynamic, 0)
	if err != nil {
		return err
	}
	t.seen = append(t.seen, [2]uint32{fixed, dynamic})
	return nil
}

func TestUsingParamsAppliesStoredOverridesAtConstructionAndUpdate(t *testing.T) {
	context, _, _ := paramsTestOwner(t)
	fixed, err := context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Del(&fixed)) })
	fixedName, dynamicName := "fixed", "dynamic"
	astTestMust(t, context.SetUIntName(&fixed, &fixedName, 7))
	child := &paramsRecordingTactic{
		primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0),
		context:         context,
		fixed:           fixedName,
		dynamic:         dynamicName,
	}
	wrapper, err := NewUsingParamsTactic(context, child, &fixed)
	astTestMust(t, err)
	tacticOwnForTest(t, wrapper)
	if len(child.seen) != 1 || child.seen[0] != [2]uint32{7, 0} {
		t.Fatal("constructor parameters", child.seen)
	}

	outer, err := context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Del(&outer)) })
	astTestMust(t, context.SetUIntName(&outer, &fixedName, 3))
	astTestMust(t, context.SetUIntName(&outer, &dynamicName, 11))
	astTestMust(t, wrapper.UpdateParams(&outer))
	if len(child.seen) != 2 || child.seen[1] != [2]uint32{7, 11} {
		t.Fatal("stored parameter precedence", child.seen)
	}
}

func TestNestedUsingParamsKeepsInnermostOverride(t *testing.T) {
	context, _, _ := paramsTestOwner(t)
	key := "fixed"
	innerParams, err := context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Del(&innerParams)) })
	astTestMust(t, context.SetUIntName(&innerParams, &key, 7))
	outerParams, err := context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, context.Del(&outerParams)) })
	astTestMust(t, context.SetUIntName(&outerParams, &key, 3))
	child := &paramsRecordingTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0), context: context, fixed: key}
	inner, err := NewUsingParamsTactic(context, child, &innerParams)
	astTestMust(t, err)
	tacticOwnForTest(t, inner)
	outer, err := NewUsingParamsTactic(context, inner, &outerParams)
	astTestMust(t, err)
	tacticOwnForTest(t, outer)
	astTestMust(t, outer.UpdateParams(&outerParams))
	if got := child.seen[len(child.seen)-1][0]; got != 7 {
		t.Fatal("outer update replaced innermost fixed parameter", got)
	}
}
