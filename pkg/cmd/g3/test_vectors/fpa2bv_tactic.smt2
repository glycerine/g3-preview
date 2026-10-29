(set-logic QF_FP)
(declare-const x (_ FloatingPoint 3 4))
(assert (fp.isNaN x))
(check-sat-using (then fpa2bv smt))
