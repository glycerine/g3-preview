// Source: src/util/vector.h:59-100,303-310,389-405.
// Licensed under the MIT license; see LICENSE.txt.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// The storage engine is shared utility code; SAT supplies its callers.
type RuntimeVectorColumn = util.RuntimeVectorColumn

var ReadRuntimeWord = util.ReadRuntimeWord
var WriteRuntimeWord = util.WriteRuntimeWord
var RuntimeByteColumnNext = util.RuntimeByteColumnNext
var RuntimeScalarColumnNext = util.RuntimeScalarColumnNext
