(set-logic UF)
(define-fun reflex ((z Bool)) Bool (forall ((y Bool)) (= z y)))
(assert (exists ((x Bool)) (reflex x)))
(check-sat)
