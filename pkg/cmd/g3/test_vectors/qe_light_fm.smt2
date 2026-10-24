(set-logic ALL)
(assert (exists ((x Real) (y Real))
  (and (<= (+ (* (- 1.0) x) y) 0.0) (<= x 2.0) (>= y 3.0))))
(check-sat-using (then simplify qe-light))
