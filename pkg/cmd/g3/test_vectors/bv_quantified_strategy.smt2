(set-logic BV)
(assert (exists ((x (_ BitVec 4))) (not (= x x))))
(check-sat)
(check-sat-using bv)
