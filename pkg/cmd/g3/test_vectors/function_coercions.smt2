(set-logic ALL)
(declare-fun f (Real) Real)
(assert (distinct (f 2) (f 2.0)))
(check-sat)
