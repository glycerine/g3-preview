// Source: src/smt/smt_context.cpp check_preamble/check_finalize,
// check/setup_and_check early-conflict branches and init_search proof reset.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (k *SMTKernel) aboveMemoryWatermark() (bool, error) {
	runtime := k.manager.NumeralService().RuntimeOwner()
	if runtime == nil {
		return false, nil
	}
	if process := runtime.ProcessOwner(); process != nil {
		return process.AboveWatermark(), nil
	}
	return runtime.AboveWatermark()
}

func (k *SMTKernel) beginCheck() (bool, error) {
	if err := k.open(); err != nil {
		return false, err
	}
	if err := k.clearLiteralResults(); err != nil {
		return false, err
	}
	if err := k.clearGeneratedModel(); err != nil {
		return false, err
	}
	if err := k.clearUnsatCore(); err != nil {
		return false, err
	}
	k.lastFailure = SMTFailureOK
	k.labels = nil
	k.internalizer.statistics.NumChecks++
	// Source m_unsat_proof is reset by init_search, not by check. A known
	// base conflict therefore retains its original proof across checks.
	above, err := k.aboveMemoryWatermark()
	if err != nil {
		return false, err
	}
	if above {
		k.lastFailure = SMTFailureMemory
		return false, nil
	}
	return true, nil
}

func (k *SMTKernel) finalizeCheck(result lbool.LBool, resultErr error) (lbool.LBool, error) {
	if resultErr != nil || result != lbool.True {
		return result, resultErr
	}
	canceled, err := k.manager.Canceled()
	if err != nil {
		return lbool.Undef, err
	}
	if canceled {
		result = lbool.Undef
		k.lastFailure = SMTFailureResourceLimit
		if k.manager.Limit.CancelFlag() {
			k.lastFailure = SMTFailureCanceled
		}
	}
	return result, nil
}

// Source VERIFY(!resolve_conflict()) builds a newly discovered base conflict's
// proof without initializing search or creating its case-split queue.
func (k *SMTKernel) resolveCheckConflict() (result lbool.LBool, resultErr error) {
	conflict, found := k.internalizer.Conflict()
	if !found {
		return lbool.Undef, ast.ErrASTDomain
	}
	resolver, err := NewSMTConflictResolver(k.internalizer)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, resolver.Close())
		if resultErr != nil {
			result = lbool.Undef
		}
	}()
	literal := k.internalizer.ConflictLiteral()
	resolution, err := resolver.Resolve(conflict, literal, k.internalizer.scopeLevel)
	if err != nil {
		return lbool.Undef, err
	}
	if resolution.Resolved {
		return lbool.Undef, ast.ErrASTDomain
	}
	proof, err := resolver.LemmaProof()
	if err != nil {
		return lbool.Undef, err
	}
	if err := k.manager.SetRef(k.unsatProof, proof); err != nil {
		return lbool.Undef, err
	}
	if err := k.captureUnsatCore(resolver, conflict, literal); err != nil {
		return lbool.Undef, err
	}
	return lbool.False, nil
}
