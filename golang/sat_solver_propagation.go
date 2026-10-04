// Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

// These raw positions capture the actual source payload/end. A moved/freed
// backing is rejected at dereference, never kept alive by an owning Go slice.
func satPropagationCursorRecord(i *satSolverWatchIterator, n uint32) (satSolverWatchListRecord, error) {
	if e := i.store.open(); e != nil {
		return satSolverWatchListRecord{}, e
	}
	if !i.store.payloads.IsLive(i.payload) || !i.store.lifetimes.IsLive(i.lifetime) {
		return satSolverWatchListRecord{}, arena.ErrInvalidSlot
	}
	x := satSolverWatchListRecord{payload: i.payload, lifetime: i.lifetime, capacity: i.capacity}
	live, e := i.store.initialized(&x, n)
	if e != nil {
		return x, e
	}
	if !live {
		return x, arena.ErrBounds
	}
	return x, nil
}
func satPropagationCursorRead(i *satSolverWatchIterator, n uint32) (satWatched, error) {
	x, e := satPropagationCursorRecord(i, n)
	if e != nil {
		return satWatched{}, e
	}
	return i.store.decode(&x, n)
}
func satPropagationCursorWrite(i *satSolverWatchIterator, n uint32, w satWatched) error {
	x, e := satPropagationCursorRecord(i, n)
	if e != nil {
		return e
	}
	return i.store.encode(&x, n, w)
}
func (s *satSolverPropagationState) propagationConflictCleanup(i *satSolverWatchIterator, from uint32, to *uint32, w satSolverWatchListView) error {
	for ; from < i.end; from++ {
		x, e := satPropagationCursorRead(i, from)
		if e != nil {
			return e
		}
		if e = satPropagationCursorWrite(i, *to, x); e != nil {
			return e
		}
		*to++
	}
	return s.finishPropagationList(i, w, *to)
}
func (s *satSolverPropagationState) propagateCore(update bool) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	if s.inconsistent {
		return false, nil
	}
	v := s.variables
	for {
		size, e := v.trail.Size(v.trailRef)
		if e != nil {
			return false, e
		}
		if v.qhead >= size {
			break
		}
		if e = s.checkpoint(); e != nil {
			return false, e
		}
		if s.cleanerCounter == -1<<31 {
			return false, ErrSATSignedCounterDomain
		}
		s.cleanerCounter--
		if satStorageSourceDebug && s.inconsistent {
			return false, ErrSATAssertion
		}
		l, e := v.trailRead(v.qhead)
		if e != nil {
			return false, e
		}
		v.qhead++
		not := l.negated()
		if satStorageSourceDebug {
			a, e := v.valueLiteral(l)
			if e != nil {
				return false, e
			}
			if a != lTrue {
				return false, ErrSATAssertion
			}
			a, e = v.valueLiteral(not)
			if e != nil {
				return false, e
			}
			if a != lFalse {
				return false, ErrSATAssertion
			}
		}
		list, e := v.watches.Get(l.index())
		if e != nil {
			return false, e
		}
		count, e := list.Size()
		if e != nil {
			return false, e
		}
		s.asymmCounter = int32(uint32(s.asymmCounter) - count)
		s.probingCounter = int32(uint32(s.probingCounter) - count)
		ref, e := list.ref()
		if e != nil {
			return false, e
		}
		iter, e := v.watches.store.Iter(ref)
		if e != nil {
			return false, e
		}
		out := uint32(0)
		for pos := uint32(0); pos < iter.end; pos++ {
			w, e := satPropagationCursorRead(&iter, pos)
			if e != nil {
				return false, e
			}
			keep := true
			switch w.Kind() {
			case 0:
				l1, e := w.Literal()
				if e != nil {
					return false, e
				}
				value, e := v.valueLiteral(l1)
				if e != nil {
					return false, e
				}
				switch value {
				case lFalse:
					if e = s.propagationConflictCleanup(&iter, pos, &out, list); e != nil {
						return false, e
					}
					s.setConflict(newSATBinaryJustification(not), l1.negated())
					return false, nil
				case lUndef:
					s.binaryPropagations++
					if e = s.assignCore(l1, newSATBinaryJustification(not)); e != nil {
						return false, e
					}
				}
			case 1:
				l1, e := w.Literal1()
				if e != nil {
					return false, e
				}
				l2, e := w.Literal2()
				if e != nil {
					return false, e
				}
				a, e := v.valueLiteral(l1)
				if e != nil {
					return false, e
				}
				b, e := v.valueLiteral(l2)
				if e != nil {
					return false, e
				}
				if a == lFalse && b == lUndef {
					s.ternaryPropagations++
					if e = s.assignCore(l2, newSATTernaryJustification(l1, not)); e != nil {
						return false, e
					}
				} else if a == lUndef && b == lFalse {
					s.ternaryPropagations++
					if e = s.assignCore(l1, newSATTernaryJustification(l2, not)); e != nil {
						return false, e
					}
				} else if a == lFalse && b == lFalse {
					if e = s.propagationConflictCleanup(&iter, pos, &out, list); e != nil {
						return false, e
					}
					s.setConflict(newSATTernaryJustification(l1, not), l2.negated())
					return false, nil
				}
			case 2:
				blocked, e := w.BlockedLiteral()
				if e != nil {
					return false, e
				}
				value, e := v.valueLiteral(blocked)
				if e != nil {
					return false, e
				}
				if value == lTrue {
					break
				}
				off, e := w.ClauseOffset()
				if e != nil {
					return false, e
				}
				c, e := s.clauses.GetClause(off)
				if e != nil {
					return false, e
				}
				c0, e := s.clauses.Literal(c, 0)
				if e != nil {
					return false, e
				}
				if c0 == not {
					c1, e := s.clauses.Literal(c, 1)
					if e != nil {
						return false, e
					}
					if e = s.clauses.SetLiteral(c, 0, c1); e != nil {
						return false, e
					}
					if e = s.clauses.SetLiteral(c, 1, c0); e != nil {
						return false, e
					}
				}
				removed, e := s.clauses.Removed(c)
				if e != nil {
					return false, e
				}
				if removed {
					break
				}
				c1, e := s.clauses.Literal(c, 1)
				if e != nil {
					return false, e
				}
				if c1 != not {
					break
				}
				c0, e = s.clauses.Literal(c, 0)
				if e != nil {
					return false, e
				}
				value, e = v.valueLiteral(c0)
				if e != nil {
					return false, e
				}
				if value == lTrue {
					w.SetClause(c0, off)
					break
				}
				n, e := s.clauses.Size(c)
				if e != nil {
					return false, e
				}
				moved := false
				for j := uint32(2); j < n; j++ {
					replacement, e := s.clauses.Literal(c, j)
					if e != nil {
						return false, e
					}
					value, e := v.valueLiteral(replacement)
					if e != nil {
						return false, e
					}
					if value == lFalse {
						continue
					}
					if e = s.clauses.SetLiteral(c, 1, replacement); e != nil {
						return false, e
					}
					if e = s.clauses.SetLiteral(c, j, not); e != nil {
						return false, e
					}
					dest, e := v.watches.Get(replacement.negated().index())
					if e != nil {
						return false, e
					}
					nw, e := newSATClauseWatch(c0, off)
					if e != nil {
						return false, e
					}
					if e = dest.Push(nw); e != nil {
						return false, e
					}
					moved = true
					keep = false
					break
				}
				if moved {
					break
				}
				value, e = v.valueLiteral(c0)
				if e != nil {
					return false, e
				}
				if satStorageSourceDebug && value != lFalse && value != lUndef {
					return false, ErrSATAssertion
				}
				if value == lFalse {
					if e = s.clauses.MarkUsed(c); e != nil {
						return false, e
					}
					if e = s.propagationConflictCleanup(&iter, pos, &out, list); e != nil {
						return false, e
					}
					s.setConflict(newSATClauseJustification(off), satDefaultLiteral())
					return false, nil
				}
				// Source retains this watch BEFORE increment/mark/assignment/glue actions.
				if e = satPropagationCursorWrite(&iter, out, w); e != nil {
					return false, e
				}
				out++
				keep = false
				s.clausePropagations++
				if e = s.clauses.MarkUsed(c); e != nil {
					return false, e
				}
				if e = s.assignCore(c0, newSATClauseJustification(off)); e != nil {
					return false, e
				}
				if update {
					learned, e := s.clauses.Learned(c)
					if e != nil {
						return false, e
					}
					if learned {
						glue, e := s.clauses.Glue(c)
						if e != nil {
							return false, e
						}
						if glue > 2 {
							g, e := s.numDiffLevels(c)
							if e != nil {
								return false, e
							}
							if g+1 < glue {
								if e = s.clauses.SetGlue(c, g); e != nil {
									return false, e
								}
							}
						}
					}
				}
			case 3:
				if satStorageSourceDebug && !s.extensionPresent {
					return false, ErrSATAssertion
				}
				return false, ErrSATMissingSourceExtBinding
			default:
				return false, ErrSATAssertion
			}
			if keep {
				if e = satPropagationCursorWrite(&iter, out, w); e != nil {
					return false, e
				}
				out++
			}
		}
		if e = s.finishPropagationList(&iter, list, out); e != nil {
			return false, e
		}
	}
	if satStorageSourceDebug {
		n, e := v.trail.Size(v.trailRef)
		if e != nil {
			return false, e
		}
		if v.qhead != n || s.inconsistent {
			return false, ErrSATAssertion
		}
	}
	return true, nil
}

// Source set_end subtracts the captured pointer from the current list backing.
// Validate that relation at this action, including the no-more-reads path.
func (s *satSolverPropagationState) finishPropagationList(i *satSolverWatchIterator, w satSolverWatchListView, n uint32) error {
	r, e := w.ref()
	if e != nil {
		return e
	}
	x, e := i.store.record(r)
	if e != nil {
		return e
	}
	if x.payload != i.payload || x.lifetime != i.lifetime {
		return arena.ErrInvalidSlot
	}
	return w.SetEnd(n)
}
