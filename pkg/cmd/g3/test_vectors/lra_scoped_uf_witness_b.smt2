; Fixed differential corpus case uf-43.
; Base witness: x=-2, y=-1, z=-1, b=true.
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
(assert (> (+ (* 2 x) (* 3 y) (* (- 3) z) 0) (- 7)))
(assert (> (+ (* 1 x) (* 1 z) (- (/ 1 3))) (- (/ 16 3))))
(assert (> (+ (* (- 3) x) (* 3 y) (* (- 3) z) 3) 8))
(assert (> (+ (* (- 2) x) (* 1 y) (* 3 z) 0) (- 1)))
(assert (<= (+ (* (- 1) x) (* 1 y) (* (- 2) z) 0) 3))
(assert (<= (+ (* (- 3) x) (* 3 y) (* (- 2) z) (- (/ 1 2))) (/ 9 2)))
(assert (> (+ (* 2 x) (* 2 y) (* 2 z) (- 3)) (- 12)))
(assert (> (+ (* (- 1) x) (* (- 1) y) (* 1 z) (/ 2 3)) (/ 5 3)))
(assert (or (>= x (- 2)) (>= y 1)))
(assert (or b (< z 0)))
(assert (= (f x) y))
(assert (distinct (f x) (f z)))
(check-sat)
(get-value (
  (and (> (+ (* 2 x) (* 3 y) (* (- 3) z) 0) (- 7)) (> (+ (* 1 x) (* 1 z) (- (/ 1 3))) (- (/ 16 3))) (> (+ (* (- 3) x) (* 3 y) (* (- 3) z) 3) 8) (> (+ (* (- 2) x) (* 1 y) (* 3 z) 0) (- 1)) (<= (+ (* (- 1) x) (* 1 y) (* (- 2) z) 0) 3) (<= (+ (* (- 3) x) (* 3 y) (* (- 2) z) (- (/ 1 2))) (/ 9 2)) (> (+ (* 2 x) (* 2 y) (* 2 z) (- 3)) (- 12)) (> (+ (* (- 1) x) (* (- 1) y) (* 1 z) (/ 2 3)) (/ 5 3)) (or (>= x (- 2)) (>= y 1)) (or b (< z 0)) (= (f x) y) (distinct (f x) (f z)))
  (not (and (> (+ (* 2 x) (* 3 y) (* (- 3) z) 0) (- 7)) (> (+ (* 1 x) (* 1 z) (- (/ 1 3))) (- (/ 16 3))) (> (+ (* (- 3) x) (* 3 y) (* (- 3) z) 3) 8) (> (+ (* (- 2) x) (* 1 y) (* 3 z) 0) (- 1)) (<= (+ (* (- 1) x) (* 1 y) (* (- 2) z) 0) 3) (<= (+ (* (- 3) x) (* 3 y) (* (- 2) z) (- (/ 1 2))) (/ 9 2)) (> (+ (* 2 x) (* 2 y) (* 2 z) (- 3)) (- 12)) (> (+ (* (- 1) x) (* (- 1) y) (* 1 z) (/ 2 3)) (/ 5 3)) (or (>= x (- 2)) (>= y 1)) (or b (< z 0)) (= (f x) y) (distinct (f x) (f z))))))
(push 1)
(assert (>= x (- 1)))
(assert (>= y 0))
(assert (>= z 0))
(assert (<= (+ x y z) (- 4)))
(check-sat)
(pop 1)
(check-sat)
(get-value (
  (and (> (+ (* 2 x) (* 3 y) (* (- 3) z) 0) (- 7)) (> (+ (* 1 x) (* 1 z) (- (/ 1 3))) (- (/ 16 3))) (> (+ (* (- 3) x) (* 3 y) (* (- 3) z) 3) 8) (> (+ (* (- 2) x) (* 1 y) (* 3 z) 0) (- 1)) (<= (+ (* (- 1) x) (* 1 y) (* (- 2) z) 0) 3) (<= (+ (* (- 3) x) (* 3 y) (* (- 2) z) (- (/ 1 2))) (/ 9 2)) (> (+ (* 2 x) (* 2 y) (* 2 z) (- 3)) (- 12)) (> (+ (* (- 1) x) (* (- 1) y) (* 1 z) (/ 2 3)) (/ 5 3)) (or (>= x (- 2)) (>= y 1)) (or b (< z 0)) (= (f x) y) (distinct (f x) (f z)))
  (not (and (> (+ (* 2 x) (* 3 y) (* (- 3) z) 0) (- 7)) (> (+ (* 1 x) (* 1 z) (- (/ 1 3))) (- (/ 16 3))) (> (+ (* (- 3) x) (* 3 y) (* (- 3) z) 3) 8) (> (+ (* (- 2) x) (* 1 y) (* 3 z) 0) (- 1)) (<= (+ (* (- 1) x) (* 1 y) (* (- 2) z) 0) 3) (<= (+ (* (- 3) x) (* 3 y) (* (- 2) z) (- (/ 1 2))) (/ 9 2)) (> (+ (* 2 x) (* 2 y) (* 2 z) (- 3)) (- 12)) (> (+ (* (- 1) x) (* (- 1) y) (* 1 z) (/ 2 3)) (/ 5 3)) (or (>= x (- 2)) (>= y 1)) (or b (< z 0)) (= (f x) y) (distinct (f x) (f z))))))
