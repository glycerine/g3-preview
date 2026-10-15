// Source: src/api/api_config_params.cpp:69-94.
package api

import (
	"errors"

	"github.com/glycerine/g3/pkg/smtcore"
)

var ErrConfigOwner = errors.New("api: invalid or closed config")

// Config owns the concrete context_params object returned by Z3_mk_config.
type Config struct {
	params *smtcore.ContextParams
}

// MkConfig initializes the process and constructs context_params from its
// current global parameter state.
func MkConfig() (*Config, error) {
	g, err := globalParamService()
	if err != nil {
		return nil, err
	}
	p, err := smtcore.NewContextParams(g.Context, g, g.Config.Budget)
	if err != nil {
		if p != nil {
			return nil, errors.Join(err, p.Close())
		}
		return nil, err
	}
	return &Config{params: p}, nil
}

// Close is the owning Z3_del_config operation.
func (c *Config) Close() error {
	if c == nil || c.params == nil {
		return ErrConfigOwner
	}
	if err := c.params.Close(); err != nil {
		return err
	}
	c.params = nil
	return nil
}

// SetParamValue retains Z3_set_param_value's catch: a project exception is
// reported through the process warning stream and does not escape this call.
func SetParamValue(c *Config, id, value string) error {
	if c == nil || c.params == nil {
		return ErrConfigOwner
	}
	return globalParamCaughtWarning(c.params.Set(&id, &value))
}
