// Source: src/sat/sat_solver.cpp:2071-2084; src/util/util.h:176-186.
package sat

import "github.com/glycerine/g3/pkg/util/diagnostic"

func (s *satSolverConflictResolution) emitCoreUpdateNotice(minSize, coreSize uint32) error {
	enabled, err := diagnostic.ProcessVerboseEnabled(1)
	if err != nil || !enabled {
		return err
	}
	out, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		return err
	}
	if err = out.AppendCString("(sat.updating core "); err != nil {
		return err
	}
	if err = out.AppendUint32(minSize); err != nil {
		return err
	}
	if err = out.AppendCString(" "); err != nil {
		return err
	}
	if err = out.AppendUint32(coreSize); err != nil {
		return err
	}
	return out.AppendCString(")\n")
}

func (s *satSolverConflictResolution) emitCoreReport() error {
	enabled, err := diagnostic.ProcessVerboseEnabled(2)
	if err != nil || !enabled {
		return err
	}
	out, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		return err
	}
	if err = out.AppendCString("(sat.core: "); err != nil {
		return err
	}
	n, err := s.core.Size()
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			if err = out.AppendCString(" "); err != nil {
				return err
			}
		}
		lit, readErr := s.core.Read(i)
		if readErr != nil {
			return readErr
		}
		value := Literal(lit)
		if value.sign() {
			if err = out.AppendCString("-"); err != nil {
				return err
			}
		}
		if err = out.AppendUint32(uint32(value.variable())); err != nil {
			return err
		}
	}
	return out.AppendCString(")\n")
}
