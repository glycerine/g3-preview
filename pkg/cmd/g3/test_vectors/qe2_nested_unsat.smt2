(set-logic ALL)
(assert (not (forall ((x Bool)) (exists ((y Bool)) (= x y)))))
(check-sat-using (then qe2 smt))
