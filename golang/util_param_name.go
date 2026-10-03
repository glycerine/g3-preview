// Copyright (c) 2011 Microsoft Corporation
// Source: src/util/params.cpp norm_param_name overloads.
package g3

import (
	"strings"

	"github.com/glycerine/g3/golang/arena"
)

func paramNameByte(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value - 'A' + 'a'
	}
	if value == '-' || value == ':' {
		return '_'
	}
	return value
}

// NormParamName preserves the nullable C-string and single leading colon.
// Only ASCII capitals, hyphen and colon change; UTF-8 bytes are not case-folded.
// The returned string belongs to the caller, not a retained parameter owner.
func NormParamName(text *string) string {
	if text == nil {
		return "_"
	}
	input := *text
	if end := strings.IndexByte(input, 0); end >= 0 {
		input = input[:end]
	}
	if len(input) != 0 && input[0] == ':' {
		input = input[1:]
	}
	if len(input) == 0 {
		return "_"
	}
	var output strings.Builder
	output.Grow(len(input))
	for i := 0; i < len(input); i++ {
		output.WriteByte(paramNameByte(input[i]))
	}
	return output.String()
}

// NormParamSymbolName reads the live owner's bare C-string directly. It does
// not allocate an intermediate spelling string or intern the normalized result.
func NormParamSymbolName(manager *SymbolManager, name Symbol) (string, error) {
	if manager == nil {
		return "", ErrInvalidSymbol
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if err := manager.checkPlain(name); err != nil {
		return "", err
	}
	if name.IsNumerical() {
		return "", ErrInvalidSymbol
	}
	if name.value == 0 {
		return "_", nil
	}
	key, err := manager.key(name)
	if err != nil {
		return "", err
	}
	length, start := key.Length(), uint32(0)
	if length != 0 {
		first, err := manager.region.Read(key.Allocation(), key.Offset())
		if err != nil {
			return "", err
		}
		if first == ':' {
			start = 1
		}
	}
	if start == length {
		return "_", nil
	}
	if uint64(length-start) > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	var output strings.Builder
	output.Grow(int(length - start))
	for i := start; i < length; i++ {
		value, err := manager.region.Read(key.Allocation(), key.Offset()+i)
		if err != nil {
			return "", err
		}
		output.WriteByte(paramNameByte(value))
	}
	return output.String(), nil
}
