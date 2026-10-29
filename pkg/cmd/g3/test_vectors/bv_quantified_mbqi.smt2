(set-logic BV)
(assert (forall ((x (_ BitVec 4))) (= x #b0000)))
(check-sat)
