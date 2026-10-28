(set-logic ALL)
(declare-const c (_ BitVec 8))
(simplify (seq.in.re (seq.unit c) (re.inter (seq.to.re (seq.unit c)) (seq.to.re (seq.unit #x61)))))
