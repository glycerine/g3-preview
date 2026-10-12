// Source: src/util/env_params.cpp:25-33; src/util/util.h:391-399.
package params

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

// EnvParamsRuntime binds the source process-global update to its concrete Go
// owners. It borrows all three owners and retains only the warning-enable bit.
type EnvParamsRuntime struct {
	globals  *GlobalParams
	verbose  *diagnostic.VerboseContext
	memory   *memory.Runtime
	warnings bool
}

func NewEnvParamsRuntime(g *GlobalParams, v *diagnostic.VerboseContext, m *memory.Runtime) (*EnvParamsRuntime, error) {
	if g == nil || v == nil || m == nil {
		return nil, arena.ErrConfig
	}
	return &EnvParamsRuntime{globals: g, verbose: v, memory: m, warnings: true}, nil
}

func (e *EnvParamsRuntime) WarningsEnabled() bool { return e != nil && e.warnings }

// Update follows the five source statements. A later failed getter or setter
// leaves earlier assignments visible, as env_params::updt_params does.
func (e *EnvParamsRuntime) Update() error {
	if e == nil || e.globals == nil || e.verbose == nil || e.memory == nil {
		return arena.ErrConfig
	}
	p, err := e.globals.GetRef()
	if err != nil {
		return err
	}
	defaultVerbosity, err := e.verbose.VerbosityLevel()
	if err != nil {
		return err
	}
	verbose := "verbose"
	level, err := e.globals.ContextField.GetUIntName(p, &verbose, defaultVerbosity)
	if err != nil {
		return err
	}
	if err = e.verbose.SetVerbosityLevel(level); err != nil {
		return err
	}
	warning := "warning"
	enabled, err := e.globals.ContextField.GetBoolName(p, &warning, true)
	if err != nil {
		return err
	}
	e.warnings = enabled
	maxSize := "memory_max_size"
	megabytes, err := e.globals.ContextField.GetUIntName(p, &maxSize, 0)
	if err != nil {
		return err
	}
	if err = e.memory.SetMaxSizeMegabytes(megabytes); err != nil {
		return err
	}
	maxCount := "memory_max_alloc_count"
	count, err := e.globals.ContextField.GetUIntName(p, &maxCount, 0)
	if err != nil {
		return err
	}
	if err = e.memory.SetMaxCount(uint64(count)); err != nil {
		return err
	}
	watermark := "memory_high_watermark"
	mark, err := e.globals.ContextField.GetUIntName(p, &watermark, 0)
	if err != nil {
		return err
	}
	return e.memory.SetWatermark(uint64(mark))
}
