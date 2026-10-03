//go:build !g3debug

// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/mpnum"

func (m *SexprManager) sexprBVDiagnostic(c *mpnum.OwnedRational, base int64) error { return nil }
