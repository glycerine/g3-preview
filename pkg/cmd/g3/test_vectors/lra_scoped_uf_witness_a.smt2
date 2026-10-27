; Fixed differential corpus case uf-7.
; Base witness: x=1, y=2/3, z=2, b=true.
; Assign f(x)=y and f(z)=y+1 at this witness; x and z differ by one.
; The child bounds contradict the sum bound. Check the base model before
; push and after pop without requiring the two solvers to choose equal values.
(set-option :produce-models true)
(set-logic QF_UFLRA)
(declare-const x Real)
(declare-const y Real)
(declare-const z Real)
(declare-const b Bool)
(declare-fun f (Real) Real)
(assert (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 1 3))) (- (/ 5 3))))
(assert (<= (+ (* (- 3) x) (- 3)) (- 3)))
(assert (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 2 3))) 0))
(assert (< (+ (* (- 2) x) (* (- 3) y) (* 1 z) 3) 3))
(assert (>= (+ (* 1 x) (* (- 3) y) (* 1 z) (- (/ 1 3))) (/ 2 3)))
(assert (<= (+ (* 1 x) (* 2 y) (* (- 2) z) (- 1)) (- (/ 8 3))))
(assert (< (+ (* (- 2) x) (* 2 z) (/ 1 2)) (/ 19 6)))
(assert (>= (+ (* (- 1) y) (* (- 1) z) (- 2)) (- (/ 14 3))))
(assert (or (>= x 1) (>= y (/ 8 3))))
(assert (or b (< z 3)))
(assert (= (f x) y))
(assert (distinct (f x) (f z)))
(check-sat)
(get-value (
  (and (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 1 3))) (- (/ 5 3))) (<= (+ (* (- 3) x) (- 3)) (- 3)) (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 2 3))) 0) (< (+ (* (- 2) x) (* (- 3) y) (* 1 z) 3) 3) (>= (+ (* 1 x) (* (- 3) y) (* 1 z) (- (/ 1 3))) (/ 2 3)) (<= (+ (* 1 x) (* 2 y) (* (- 2) z) (- 1)) (- (/ 8 3))) (< (+ (* (- 2) x) (* 2 z) (/ 1 2)) (/ 19 6)) (>= (+ (* (- 1) y) (* (- 1) z) (- 2)) (- (/ 14 3))) (or (>= x 1) (>= y (/ 8 3))) (or b (< z 3)) (= (f x) y) (distinct (f x) (f z)))
  (not (and (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 1 3))) (- (/ 5 3))) (<= (+ (* (- 3) x) (- 3)) (- 3)) (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 2 3))) 0) (< (+ (* (- 2) x) (* (- 3) y) (* 1 z) 3) 3) (>= (+ (* 1 x) (* (- 3) y) (* 1 z) (- (/ 1 3))) (/ 2 3)) (<= (+ (* 1 x) (* 2 y) (* (- 2) z) (- 1)) (- (/ 8 3))) (< (+ (* (- 2) x) (* 2 z) (/ 1 2)) (/ 19 6)) (>= (+ (* (- 1) y) (* (- 1) z) (- 2)) (- (/ 14 3))) (or (>= x 1) (>= y (/ 8 3))) (or b (< z 3)) (= (f x) y) (distinct (f x) (f z))))))
(push 1)
(assert (>= x 2))
(assert (>= y (/ 5 3)))
(assert (>= z 3))
(assert (<= (+ x y z) (/ 11 3)))
(check-sat)
(pop 1)
(check-sat)
(get-value (
  (and (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 1 3))) (- (/ 5 3))) (<= (+ (* (- 3) x) (- 3)) (- 3)) (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 2 3))) 0) (< (+ (* (- 2) x) (* (- 3) y) (* 1 z) 3) 3) (>= (+ (* 1 x) (* (- 3) y) (* 1 z) (- (/ 1 3))) (/ 2 3)) (<= (+ (* 1 x) (* 2 y) (* (- 2) z) (- 1)) (- (/ 8 3))) (< (+ (* (- 2) x) (* 2 z) (/ 1 2)) (/ 19 6)) (>= (+ (* (- 1) y) (* (- 1) z) (- 2)) (- (/ 14 3))) (or (>= x 1) (>= y (/ 8 3))) (or b (< z 3)) (= (f x) y) (distinct (f x) (f z)))
  (not (and (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 1 3))) (- (/ 5 3))) (<= (+ (* (- 3) x) (- 3)) (- 3)) (<= (+ (* 3 x) (* 1 y) (* (- 3) z) (- (/ 2 3))) 0) (< (+ (* (- 2) x) (* (- 3) y) (* 1 z) 3) 3) (>= (+ (* 1 x) (* (- 3) y) (* 1 z) (- (/ 1 3))) (/ 2 3)) (<= (+ (* 1 x) (* 2 y) (* (- 2) z) (- 1)) (- (/ 8 3))) (< (+ (* (- 2) x) (* 2 z) (/ 1 2)) (/ 19 6)) (>= (+ (* (- 1) y) (* (- 1) z) (- 2)) (- (/ 14 3))) (or (>= x 1) (>= y (/ 8 3))) (or b (< z 3)) (= (f x) y) (distinct (f x) (f z))))))
