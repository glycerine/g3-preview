// Source: src/util/stopwatch.h.
// Translation of src/util/stopwatch.h's POSIX control flow.
package util

// Stopwatch is cold report metadata. Source reset clears only accumulated time;
// querying a running watch performs stop then start. Clock reads can fail in Go
// instead of using an uninitialized POSIX timespec after a system-call failure.
type Stopwatch struct {
	elapsed, start uint64
	running        bool
	clock          func() (uint64, error) // nil selects the actual platform clock.
}

func (w *Stopwatch) read() (uint64, error) {
	if w.clock != nil {
		return w.clock()
	}
	return platformNanoseconds()
}
func (w *Stopwatch) Start() error {
	if w.running {
		return nil
	}
	n, e := w.read()
	if e != nil {
		return e
	}
	w.start = n
	w.running = true
	return nil
}
func (w *Stopwatch) Stop() error {
	if !w.running {
		return nil
	}
	n, e := w.read()
	if e != nil {
		return e
	}
	// Preserve Linux's unsigned arithmetic and nanosecond borrow guard.
	w.elapsed += (n/1e9 - w.start/1e9) * 1e9
	if w.elapsed != 0 || n%1e9 >= w.start%1e9 {
		w.elapsed += n%1e9 - w.start%1e9
	}
	w.running = false
	return nil
}
func (w *Stopwatch) Reset()               { w.elapsed = 0 }
func (w *Stopwatch) Add(other *Stopwatch) { w.elapsed += other.elapsed }
func (w *Stopwatch) Seconds() (float64, error) {
	if w.running {
		if e := w.Stop(); e != nil {
			return 0, e
		}
		if e := w.Start(); e != nil {
			return 0, e
		}
	}
	return float64(w.elapsed) / 1e9, nil
}
func (w *Stopwatch) Running() bool              { return w.running }
func (w *Stopwatch) ElapsedNanoseconds() uint64 { return w.elapsed }
