(set-logic ALL)
(assert (forall ((x Bool)) x))
(check-sat-using der)
