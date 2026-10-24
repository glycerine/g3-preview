(set-logic QF_LIA)
(set-option :produce-models true)
(set-option :produce-unsat-cores true)
(declare-const x Int)
(declare-const y Int)
(assert (! (>= (+ x y) 2) :named lo))
(assert (! (<= (+ x y) 1) :named hi))
(check-sat)

