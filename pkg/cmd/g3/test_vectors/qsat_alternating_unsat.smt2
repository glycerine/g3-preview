(set-logic ALL)
(assert (exists ((x Bool)) (forall ((y Bool)) (= x y))))
(check-sat-using qsat)
