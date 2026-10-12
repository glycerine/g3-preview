//go:build !linux

// Source: src/test/lp.cpp:3234-3236, absent _LINUX_ call.
package lp

func (d *LPTestDriver) runBatch() error { return nil }
