// Source: src/smt/asserted_formulas.cpp reduce/invoke/display_ll,
// get_total_size and check_well_sorted.
package smtcore

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (p assertedPass) sourceName() string {
	return [...]string{
		"propagate-values", "find-macros", "nnf-cnf", "reduce-asserted",
		"pull-nested-quantifiers", "lift-ite", "lift-ite", "elim-term-ite",
		"refine-injectivity", "distribute-forall", "find-quasi-macros",
		"propagate-bit-vector-over-integers", "cheap-fourier-motzkin",
		"pattern-inference", "maximizing-bv-sharing",
	}[p]
}

func assertedVerbose(level uint32, text string) error {
	enabled, err := diagnostic.ProcessVerboseEnabled(level)
	if err != nil || !enabled {
		return err
	}
	out, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		return err
	}
	if err := out.AppendCString(text); err != nil {
		return err
	}
	return out.Flush()
}

func (k *SMTKernel) assertedTotalSize() (size uint32, result error) {
	visited, err := ast.NewASTExprMark(k.manager)
	if err != nil {
		return 0, err
	}
	defer func() { result = errors.Join(result, visited.Close()) }()
	for _, row := range k.assertions {
		n, err := k.manager.GetNumExprsWithExprMark(row.Formula, visited)
		if err != nil {
			return 0, err
		}
		size += n
	}
	return size, nil
}

func (k *SMTKernel) assertedVerboseSize() error {
	enabled, err := diagnostic.ProcessVerboseEnabled(10000)
	if err != nil || !enabled {
		return err
	}
	n, err := k.assertedTotalSize()
	if err != nil {
		return err
	}
	out, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		return err
	}
	if err := out.AppendCString("total size: "); err != nil {
		return err
	}
	if err := out.AppendUint32(n); err != nil {
		return err
	}
	if err := out.AppendCString("\n"); err != nil {
		return err
	}
	return out.Flush()
}

// DisplayAssertedFormulasLowLevel shares source definition visitation across
// all assertion roots. Empty managers produce no output.
func (k *SMTKernel) DisplayAssertedFormulasLowLevel(out *utext.NumericTextStream) error {
	if err := k.open(); err != nil {
		return err
	}
	if len(k.assertions) == 0 {
		return nil
	}
	roots := make([]ast.ASTHandle, len(k.assertions))
	for i, row := range k.assertions {
		roots[i] = row.Formula
	}
	if err := k.manager.DisplayLowLevelExpressionDefinitions(out, roots); err != nil {
		return err
	}
	if err := out.AppendCString("asserted formulas:\n"); err != nil {
		return err
	}
	for _, row := range k.assertions {
		id, err := k.manager.GetASTID(row.Formula)
		if err != nil {
			return err
		}
		if err := out.AppendCString("#"); err != nil {
			return err
		}
		if err := out.AppendUint32(id); err != nil {
			return err
		}
		if err := out.AppendCString(" "); err != nil {
			return err
		}
	}
	return out.AppendCString("\n")
}

// Source ostream failure is sticky; later pretty-printer operands still run.
type assertedTraceWriter struct{ out io.Writer }

func (w assertedTraceWriter) Write(p []byte) (int, error) {
	_, _ = w.out.Write(p)
	return len(p), nil
}

func (k *SMTKernel) traceAssertedState(tag, function string, line int, lowLevel bool) error {
	tags := k.modelConfig.Trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	err := tags.EmitConditional(tag, function, "src/smt/asserted_formulas.cpp", line, func() bool { return true }, func(out io.Writer) (result error) {
		out = assertedTraceWriter{out}
		if !lowLevel {
			return k.DisplayAssertedFormulas(out)
		}
		stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: k.manager.Budget()})
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, stream.Close()) }()
		if err := k.DisplayAssertedFormulasLowLevel(stream); err != nil {
			return err
		}
		text, err := stream.String()
		if err != nil {
			return err
		}
		_, err = io.WriteString(out, text)
		return err
	})
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}

func (k *SMTKernel) traceAssertedMacros() error {
	tags := k.modelConfig.Trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	err := tags.EmitConditional("macros", "reduce", "src/smt/asserted_formulas.cpp", 265, func() bool { return true }, func(out io.Writer) error {
		out = assertedTraceWriter{out}
		printer, err := ast.NewLegacySMTPrinter(k.manager)
		if err != nil {
			return err
		}
		for i := 0; i < k.assertedMacros.NumMacros(); i++ {
			d, err := k.assertedMacros.MacroDeclaration(i)
			if err != nil {
				return err
			}
			q, found := k.assertedMacros.MacroQuantifier(d)
			if !found {
				return ast.ErrASTHandle
			}
			head, def, err := k.assertedMacros.GetHeadDefinition(q, d)
			if err != nil {
				return err
			}
			if err := printer.DisplayAST(out, head, 0, nil); err != nil {
				return err
			}
			_, _ = io.WriteString(out, " ->\n")
			if err := printer.DisplayAST(out, def, 0, nil); err != nil {
				return err
			}
			_, _ = io.WriteString(out, "\n")
		}
		return nil
	})
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}

func (k *SMTKernel) checkAssertedWellSorted(line int32) error {
	if !diagnostic.DebugBuild {
		return nil
	}
	d := diagnostic.BorrowProcessDebugContext()
	enabled, err := d.ConditionalAssertionEnabled("well_sorted")
	if err != nil || !enabled {
		return err
	}
	for _, row := range k.assertions {
		sorted, err := k.manager.IsWellSorted(row.Formula, nil)
		if err != nil {
			return err
		}
		if !sorted {
			return diagnostic.DebugAssertionFailure(d, "src/smt/asserted_formulas.cpp", line, "check_well_sorted()")
		}
	}
	return nil
}

func (k *SMTKernel) traceAssertedAfterReduce(function string, normalLine, lowLine int) error {
	if err := k.traceAssertedState("after_reduce", function, normalLine, false); err != nil {
		return err
	}
	return k.traceAssertedState("after_reduce_ll", function, lowLine, true)
}
