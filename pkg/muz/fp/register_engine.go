package fp

// Source: src/muz/fp/dl_register_engine.cpp concrete fixedpoint engine factory.

import "fmt"

// SelectEngine applies the retained factory branch. Auto-configuration selects
// Datalog for the exact positive propositional fragment; other native engines
// remain explicit until their concrete Go implementations exist.
func (e *PropositionalEngine) SelectEngine(name string) error {
	if err := e.open(); err != nil {
		return err
	}
	switch name {
	case "auto-config", "datalog":
		return nil
	case "pdr", "spacer", "bmc", "tab", "clp", "duality", "ddnf", "qpdr", "qbmc":
		return fmt.Errorf("muz/fp: fixedpoint engine %s is not ported", name)
	default:
		return fmt.Errorf("muz/fp: unknown fixedpoint engine %s", name)
	}
}
