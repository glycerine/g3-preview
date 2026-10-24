(set-logic QF_LIA)
(set-option :produce-models true)
(set-option :produce-proofs true)
(declare-const x Int)
(declare-const y Int)
(assert (>= (+ x y) 2))
(assert (<= (+ x y) 1))
(check-sat)

