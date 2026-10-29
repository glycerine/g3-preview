package spacer

// Source: src/muz/spacer/spacer_context.cpp context::check_reachability and
// context::is_reachable, using the source-default non-flexible trace policy.

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func confirmConcreteReachability(obligation *ProofObligation) (_ bool, resultErr error) {
	if obligation == nil || obligation.valid() != nil {
		return false, ErrManagerOwner
	}
	saved := obligation.level
	obligation.level = InfinityLevel
	check, err := obligation.transformer.CheckReachability(obligation)
	obligation.level = saved
	if err != nil {
		return false, err
	}
	defer func() { resultErr = errors.Join(resultErr, check.Close(obligation.transformer.manager)) }()
	if check.Status != lbool.True || !check.IsConcrete {
		return false, nil
	}
	if check.Rule != nil {
		predecessors, err := obligation.transformer.RulePredecessors(check.Rule)
		if err != nil {
			return false, err
		}
		if len(predecessors) != 0 {
			fact, err := obligation.transformer.MakeReachFact(obligation, check.Rule, check.Model, true, true)
			if err != nil {
				return false, err
			}
			_, addErr := obligation.transformer.AddReachFact(fact)
			closeErr := fact.Close()
			if addErr != nil || closeErr != nil {
				return false, errors.Join(addErr, closeErr)
			}
		}
	}
	derivation, found, err := obligation.Derivation()
	if err != nil {
		return false, err
	}
	if found {
		derivation, err = obligation.DetachDerivation()
		if err != nil {
			return false, err
		}
	}
	if err = obligation.Close(); err != nil {
		if derivation != nil {
			_ = derivation.Close()
		}
		return false, err
	}
	if derivation == nil {
		return true, nil
	}
	next, err := derivation.CreateNextChild()
	if err != nil {
		return false, errors.Join(err, derivation.Close())
	}
	if next == nil {
		return true, derivation.Close()
	}
	if err = next.SetDerivation(derivation); err != nil {
		return false, errors.Join(err, derivation.Close())
	}
	return confirmConcreteReachability(next)
}

// CheckReachability runs one source-default obligation-queue search at the
// queue's current maximum level. True means a concrete trace reached the root;
// false means the root was blocked at this level.
func (q *ProofObligationQueue) CheckReachability() (bool, error) {
	if q == nil || q.root == nil || q.root.valid() != nil {
		return false, ErrManagerOwner
	}
	if err := q.Reset(); err != nil {
		return false, err
	}
	for {
		node, err := q.Top()
		if err != nil {
			return false, err
		}
		if node == nil {
			return false, nil
		}
		closed, err := node.IsClosed()
		if err != nil {
			return false, err
		}
		dirty, err := node.Dirty()
		if err != nil {
			return false, err
		}
		if closed || dirty {
			if _, err = q.Pop(); err != nil {
				return false, err
			}
			if closed {
				if q.IsRoot(node) {
					return true, nil
				}
			} else {
				if err = node.Clean(); err != nil {
					return false, err
				}
				if err = q.Push(node); err != nil {
					return false, err
				}
			}
			continue
		}
		expansion, err := node.transformer.ExpandObligation(node)
		if err != nil {
			return false, err
		}
		switch expansion.Status {
		case lbool.True:
			if _, err = q.Pop(); err != nil {
				return false, err
			}
			if q.IsRoot(node) {
				return true, nil
			}
			candidate := node
			for candidate.parent != nil {
				parent := candidate.parent
				reachable, err := confirmConcreteReachability(parent)
				if err != nil {
					return false, err
				}
				if !reachable {
					if closed, closeErr := parent.IsClosed(); closeErr != nil {
						return false, closeErr
					} else if !closed {
						if err = parent.BumpWeakness(); err != nil {
							return false, err
						}
					}
					break
				}
				candidate = parent
				if q.IsRoot(candidate) {
					return true, nil
				}
			}
		case lbool.False:
			if _, err = q.Pop(); err != nil {
				return false, err
			}
			if dirty, err = node.Dirty(); err != nil {
				return false, err
			} else if dirty {
				if err = node.Clean(); err != nil {
					return false, err
				}
			}
			if err = node.IncrementLevel(); err != nil {
				return false, err
			}
			if q.IsRoot(node) {
				return false, nil
			}
		case lbool.Undef:
			if expansion.Child == nil {
				return false, nil
			}
			closed, err = node.IsClosed()
			if err != nil {
				return false, err
			}
			if closed {
				top, topErr := q.Top()
				if topErr != nil {
					return false, topErr
				}
				if top == node {
					if _, err = q.Pop(); err != nil {
						return false, err
					}
				}
			}
			if err = q.Push(expansion.Child); err != nil {
				return false, err
			}
		}
	}
}
