(set-logic ALL)
(assert (exists ((x Int)) (forall ((y Int)) (>= x y))))
(check-sat-using qsat)
