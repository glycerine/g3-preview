(set-logic ALL)
(assert (exists ((x Int)) (and (= x 3) (= x 4))))
(check-sat-using qe-light)
