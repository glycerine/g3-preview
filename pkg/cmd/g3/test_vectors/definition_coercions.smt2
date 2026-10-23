(set-logic ALL)
(define-fun plus1 ((x Real)) Real (+ x 1.0))
(assert (not (= (plus1 2) 3.0)))
(check-sat)
