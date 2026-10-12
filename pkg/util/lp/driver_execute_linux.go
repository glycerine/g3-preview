//go:build linux

// Source: src/test/lp.cpp:3234-3236, _LINUX_ solve_some_mps call.
package lp

func (d *LPTestDriver) runBatch() error {
	d.Batch = NewMPSBatchDriver(d.Preparation.Arguments, d.manager, d.config)
	return d.Batch.Run()
}
