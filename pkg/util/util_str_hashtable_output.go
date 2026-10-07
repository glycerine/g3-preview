// Source: src/util/hashtable.h:581-600; src/util/str_hashtable.h:24-31.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

func (s *NumericTextStream) AppendStrHashtableCString(region *arena.BytesRegion, key StrKey) error {
	return smallObjectCStringInto(s, region, GlobalParamCString{Origin: key.Origin(), Ref: key.Allocation(), Offset: key.Offset(), Present: true})
}
