(set-logic QF_AUFLIA)
(declare-fun a () (Array Int Int))
(assert (not (= (select (store a 1 7) 1) 7)))
(check-sat)
