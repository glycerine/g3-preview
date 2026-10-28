package model_test

// Source: src/model/model_params.hpp; src/model/model_evaluator_params.hpp.

import (
	"testing"

	"github.com/glycerine/g3/pkg/model"
)

func TestGeneratedModelParameterFacadesDefaultsSnapshotsAndLocalPrecedence(t *testing.T) {
	_, _, _, context, local, config := evaluatorFixture(t)

	defaults, err := model.NewModelParams(context, config.Globals, nil)
	astTestMust(t, err)
	for name, getter := range map[string]func() (bool, error){
		"partial": defaults.Partial, "v1": defaults.V1, "v2": defaults.V2,
		"compact": defaults.Compact, "completion": defaults.Completion,
	} {
		value, err := getter()
		astTestMust(t, err)
		if value {
			t.Fatal("model default", name)
		}
	}
	astTestMust(t, defaults.Close())

	for _, key := range []string{"partial", "v1", "v2", "compact", "completion"} {
		astTestMust(t, config.Globals.SourceSetName("model."+key, "true"))
	}
	parameters, err := model.NewModelParams(context, config.Globals, local)
	astTestMust(t, err)
	key := "partial"
	astTestMust(t, context.SetBoolName(local, &key, false))
	partial, err := parameters.Partial()
	astTestMust(t, err)
	compact, err := parameters.Compact()
	astTestMust(t, err)
	if partial || !compact {
		t.Fatal("model local/global precedence", partial, compact)
	}
	copyOwner := *parameters
	if _, err = copyOwner.V1(); err == nil {
		t.Fatal("copied model facade accepted")
	}
	astTestMust(t, parameters.Close())
	if _, err = parameters.V2(); err == nil {
		t.Fatal("closed model facade accepted")
	}

	evaluatorDefaults, err := model.NewModelEvaluatorParams(context, config.Globals, nil)
	astTestMust(t, err)
	maximumMemory, err := evaluatorDefaults.MaxMemory()
	astTestMust(t, err)
	maximumSteps, err := evaluatorDefaults.MaxSteps()
	astTestMust(t, err)
	completion, err := evaluatorDefaults.Completion()
	astTestMust(t, err)
	cache, err := evaluatorDefaults.Cache()
	astTestMust(t, err)
	arrays, err := evaluatorDefaults.ArrayEqualities()
	astTestMust(t, err)
	if maximumMemory != ^uint32(0) || maximumSteps != ^uint32(0) || completion || !cache || !arrays {
		t.Fatal("evaluator defaults", maximumMemory, maximumSteps, completion, cache, arrays)
	}
	astTestMust(t, evaluatorDefaults.Close())

	astTestMust(t, config.Globals.SourceSetName("model_evaluator.max_steps", "17"))
	astTestMust(t, config.Globals.SourceSetName("model_evaluator.cache", "false"))
	evaluatorParameters, err := model.NewModelEvaluatorParams(context, config.Globals, local)
	astTestMust(t, err)
	key = "max_steps"
	astTestMust(t, context.SetUIntName(local, &key, 23))
	maximumSteps, err = evaluatorParameters.MaxSteps()
	astTestMust(t, err)
	cache, err = evaluatorParameters.Cache()
	astTestMust(t, err)
	if maximumSteps != 23 || cache {
		t.Fatal("evaluator local/global precedence", maximumSteps, cache)
	}
	astTestMust(t, evaluatorParameters.Close())
}
