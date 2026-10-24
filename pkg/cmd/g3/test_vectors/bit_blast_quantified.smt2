(set-logic BV)
(assert (exists ((x (_ BitVec 3))) (= x #b101)))
(check-sat-using (then (using-params bit-blast :blast_quant true) smt))
