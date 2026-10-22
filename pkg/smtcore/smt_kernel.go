// Source: src/smt/smt_kernel.h and src/smt/smt_kernel.cpp retained assertion,
// scope, setup, check, and context-access facade.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type SMTKernelAssertion struct {
	Formula ast.ASTHandle
	Proof   ast.ASTHandle
}

type SMTKernel struct {
	manager      *ast.ASTManager
	checker      *SMTCheckerContext
	relevancy    *SMTRelevancyPropagator
	internalizer *SMTInternalizerState
	assertions   []SMTKernelAssertion
	assertedHead int
	scopes       []int
	unsatCore    []ast.ASTHandle
	assumptions  map[SMTLiteral]ast.ASTHandle
	closed       bool
}

func NewSMTKernel(manager *ast.ASTManager, quantifiers *SMTQuantifierRegistry) (*SMTKernel, error) {
	checker, err := NewSMTCheckerContext(manager)
	if err != nil {
		return nil, err
	}
	relevancy, err := NewSMTRelevancyPropagator(checker, true)
	if err != nil {
		return nil, err
	}
	internalizer, err := NewSMTInternalizerState(manager, checker, quantifiers)
	if err != nil {
		return nil, err
	}
	if err = internalizer.SetRelevancy(relevancy); err != nil {
		return nil, errors.Join(err, internalizer.Close())
	}
	return &SMTKernel{manager: manager, checker: checker, relevancy: relevancy, internalizer: internalizer}, nil
}

func (k *SMTKernel) open() error {
	if k == nil || k.manager == nil || k.internalizer == nil || k.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (k *SMTKernel) Internalizer() (*SMTInternalizerState, error) {
	if err := k.open(); err != nil {
		return nil, err
	}
	return k.internalizer, nil
}

func (k *SMTKernel) RegisterTheory(theory SMTTheoryInternalizer) error {
	if err := k.open(); err != nil {
		return err
	}
	return k.internalizer.RegisterTheory(theory)
}

func (k *SMTKernel) Assert(formula, proof ast.ASTHandle) error {
	if err := k.open(); err != nil {
		return err
	}
	sort, err := k.manager.GetSort(formula)
	if err != nil {
		return err
	}
	boolean, err := k.manager.MkBoolSort()
	if err != nil {
		return err
	}
	if sort != boolean {
		return ast.ErrASTDomain
	}
	if !proof.IsNull() {
		if isProof, err := k.manager.IsProof(proof); err != nil || !isProof {
			if err != nil {
				return err
			}
			return ast.ErrASTDomain
		}
	}
	if proof.IsNull() {
		proofs, err := k.manager.ProofsEnabled()
		if err != nil {
			return err
		}
		if proofs {
			proof, err = k.manager.MkProofAsserted(formula)
			if err != nil {
				return err
			}
		}
	}
	limit := len(k.assertions)
	if err = k.pushAssertion(formula, proof); err == nil {
		return nil
	}
	for index := len(k.assertions); index > limit; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			err = errors.Join(err, k.manager.DecRef(assertion.Proof))
		}
		err = errors.Join(err, k.manager.DecRef(assertion.Formula))
	}
	k.assertions = k.assertions[:limit]
	return err
}

func (k *SMTKernel) Assertions() []SMTKernelAssertion {
	if k == nil {
		return nil
	}
	return append([]SMTKernelAssertion(nil), k.assertions...)
}

func (k *SMTKernel) internalizePending() error {
	for k.assertedHead < len(k.assertions) {
		assertion := k.assertions[k.assertedHead]
		if err := k.internalizer.Assert(assertion.Formula, assertion.Proof, 0); err != nil {
			return err
		}
		k.assertedHead++
	}
	return nil
}

func (k *SMTKernel) Push() error {
	if err := k.open(); err != nil {
		return err
	}
	if err := k.internalizePending(); err != nil {
		return err
	}
	if err := k.internalizer.PushScope(); err != nil {
		return err
	}
	k.scopes = append(k.scopes, len(k.assertions))
	return nil
}

func (k *SMTKernel) Pop(count uint32) error {
	if err := k.open(); err != nil {
		return err
	}
	if count == 0 || count > uint32(len(k.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(k.scopes) - int(count)
	limit := k.scopes[level]
	if err := k.internalizer.PopScopes(count); err != nil {
		return err
	}
	for index := len(k.assertions); index > limit; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			if err := k.manager.DecRef(assertion.Proof); err != nil {
				return err
			}
		}
		if err := k.manager.DecRef(assertion.Formula); err != nil {
			return err
		}
	}
	k.assertions = k.assertions[:limit]
	if k.assertedHead > limit {
		k.assertedHead = limit
	}
	k.scopes = k.scopes[:level]
	return nil
}

func (k *SMTKernel) ScopeLevel() uint32 {
	if k == nil {
		return 0
	}
	return uint32(len(k.scopes))
}

func (k *SMTKernel) prepareCheck() (*SMTActivityCaseSplitQueue, error) {
	if err := k.open(); err != nil {
		return nil, err
	}
	if err := k.internalizer.SetupTheories(); err != nil {
		return nil, fmt.Errorf("smt kernel setup theories: %w", err)
	}
	if err := k.internalizePending(); err != nil {
		return nil, fmt.Errorf("smt kernel internalize assertions: %w", err)
	}
	if err := k.internalizer.InitializeTheorySearch(); err != nil {
		return nil, fmt.Errorf("smt kernel initialize theory search: %w", err)
	}
	queue, err := NewSMTActivityCaseSplitQueue(k.internalizer)
	if err != nil {
		return nil, fmt.Errorf("smt kernel case split queue: %w", err)
	}
	k.internalizer.statistics.NumChecks++
	return queue, nil
}

func (k *SMTKernel) Check() (lbool.LBool, error) {
	if err := k.clearUnsatCore(); err != nil {
		return lbool.Undef, err
	}
	queue, err := k.prepareCheck()
	if err != nil {
		return lbool.Undef, err
	}
	return k.search(queue)
}

func (k *SMTKernel) Inconsistent() bool {
	return k != nil && k.internalizer != nil && k.internalizer.Inconsistent()
}

func (k *SMTKernel) Statistics() SMTContextStatistics {
	if k == nil || k.internalizer == nil {
		return SMTContextStatistics{}
	}
	return k.internalizer.Statistics()
}

func (k *SMTKernel) Close() error {
	if k == nil || k.manager == nil {
		return ast.ErrASTHandle
	}
	if k.closed {
		return nil
	}
	var result error
	result = errors.Join(result, k.clearUnsatCore())
	result = errors.Join(result, k.internalizer.Close())
	for index := len(k.assertions); index > 0; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			result = errors.Join(result, k.manager.DecRef(assertion.Proof))
		}
		result = errors.Join(result, k.manager.DecRef(assertion.Formula))
	}
	if result == nil {
		k.assertions = nil
		k.internalizer = nil
		k.checker = nil
		k.relevancy = nil
		k.manager = nil
		k.closed = true
	}
	return result
}
