// Source: src/util/hashtable.h:581-600; src/util/str_hashtable.h:24-31.
// Source author: Leonardo de Moura (2006-09-11; 2006-09-13).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

func (s *NumericTextStream) AppendStrHashtableCString(region *arena.BytesRegion, key StrKey) error {
	return smallObjectCStringInto(s, region, GlobalParamCString{Origin: key.Origin(), Ref: key.Allocation(), Offset: key.Offset(), Present: true})
}
