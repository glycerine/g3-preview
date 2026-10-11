// Source: src/test/lp.cpp:2486-2506,2529-2545 (MPS input branch).
package lp

import (
	"os"

	"github.com/glycerine/g3/pkg/util/numeral"
)

// LARMPSFileDriver owns the two readers and LAR solver of test_lar_on_file's
// MPS branch. The creation reader dies before the second read on success;
// failed population retains it until the partial solver has closed.
// Arguments and the manager are borrowed. This entry point selects MPS input;
// the separate source SMT reader/dispatch remains to be ported.
type LARMPSFileDriver struct {
	Name       string
	Arguments  *DriverArguments
	Solver     *LARSolver
	Runner     *LARRunDriver
	Status     Status
	Iterations uint32
	creator    *RationalMPSReader
	reader     *RationalMPSReader
	manager    *numeral.RationalManager
	config     MPSDriverConfig
	started    bool
	closed     bool
}

func NewLARMPSFileDriver(name string, args *DriverArguments, manager *numeral.RationalManager, config MPSDriverConfig) *LARMPSFileDriver {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	return &LARMPSFileDriver{Name: name, Arguments: args, manager: manager, config: config}
}

func (d *LARMPSFileDriver) Run() error {
	if d.started || d.closed {
		return ErrExactVectorPending
	}
	d.started = true
	if d.Arguments == nil {
		return ErrBounds
	}
	d.creator = OpenRationalMPSReader(d.manager, d.Name)
	d.creator.Messages = d.config.Output
	if err := d.creator.Read(); err != nil {
		return err
	}
	if d.creator.IsOK() {
		var err error
		d.Solver, err = d.creator.CreateLARSolver()
		if err != nil {
			return err
		}
	} else if err := writeDriverLine(d.config.Output, "cannot process "+d.Name); err != nil {
		return err
	}
	// Successful population has no remaining borrowed reader numerals. Source
	// create_lar_solver_from_file destroys this local reader before returning.
	if err := d.creator.Close(); err != nil {
		return err
	}
	d.creator = nil
	d.reader = OpenRationalMPSReader(d.manager, d.Name)
	d.reader.Messages = d.config.Output
	if err := d.reader.Read(); err != nil {
		return err
	}
	if d.reader.IsOK() {
		d.Runner = NewLARRunDriver(d.Solver, d.reader, d.Arguments, d.config)
		if err := d.Runner.Run(); err != nil {
			return err
		}
		d.Status, d.Iterations = d.Runner.Status, d.Solver.TotalIterations()
	}
	return d.Close()
}

func (d *LARMPSFileDriver) Close() error {
	if d.closed {
		return nil
	}
	if d.Runner != nil {
		if err := d.Runner.Close(); err != nil {
			return err
		}
		d.Runner = nil
	}
	if d.Solver != nil {
		if err := d.Solver.Close(); err != nil {
			return err
		}
		d.Solver = nil
	}
	if d.creator != nil {
		if err := d.creator.Close(); err != nil {
			return err
		}
		d.creator = nil
	}
	if d.reader != nil {
		if err := d.reader.Close(); err != nil {
			return err
		}
		d.reader = nil
	}
	d.closed = true
	return nil
}

// LARMPSInputDriver implements test_lar_solver's file-before-list dispatch for
// MPS input. Completed files close before the next starts. A failed file remains
// Current until Close; owning drivers must not be copied.
type LARMPSInputDriver struct {
	Arguments *DriverArguments
	Files     []string
	Completed int
	Current   *LARMPSFileDriver
	manager   *numeral.RationalManager
	config    MPSDriverConfig
	started   bool
	closed    bool
}

func NewLARMPSInputDriver(args *DriverArguments, manager *numeral.RationalManager, config MPSDriverConfig) *LARMPSInputDriver {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	return &LARMPSInputDriver{Arguments: args, manager: manager, config: config}
}

func (d *LARMPSInputDriver) Run() error {
	if d.started || d.closed {
		return ErrExactVectorPending
	}
	d.started = true
	if d.Arguments == nil {
		return ErrBounds
	}
	if file := d.Arguments.OptionValue("--file"); file != "" {
		d.Files = []string{file}
	} else if list := d.Arguments.OptionValue("--filelist"); list != "" {
		var err error
		d.Files, err = ReadDriverFileList(list, d.config.Output)
		if err != nil {
			return err
		}
	}
	for _, file := range d.Files {
		d.Current = NewLARMPSFileDriver(file, d.Arguments, d.manager, d.config)
		if err := d.Current.Run(); err != nil {
			return err
		}
		d.Current = nil
		d.Completed++
	}
	return d.Close()
}

func (d *LARMPSInputDriver) Close() error {
	if d.closed {
		return nil
	}
	if d.Current != nil {
		if err := d.Current.Close(); err != nil {
			return err
		}
		d.Current = nil
	}
	d.closed = true
	return nil
}
