package polynomial

import "reflect"

// Preserve the old field-by-field snapshot check after scopes gained embedded
// callback hooks. These scopes use Policy; any nonnil function hook fails the
// snapshot comparison rather than treating different closures as equal.
func testNumericScopeEqual(a, b any) bool {
	return testNumericScopeValueEqual(reflect.ValueOf(a), reflect.ValueOf(b))
}
func testNumericScopeValueEqual(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	if a.Kind() == reflect.Func {
		return a.IsNil() && b.IsNil()
	}
	if a.Comparable() && b.Comparable() {
		return a.Equal(b)
	}
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !testNumericScopeValueEqual(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			if !testNumericScopeValueEqual(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return testNumericScopeValueEqual(a.Elem(), b.Elem())
	}
	return false
}
