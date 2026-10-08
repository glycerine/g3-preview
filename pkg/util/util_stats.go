// Source: src/util/stats.h:25-35.
package util

import (
	"io"
	"strconv"
	"strings"
)

func statMessagePrefix(message string) string {
	if end := strings.IndexByte(message, 0); end >= 0 {
		return message[:end]
	}
	return message
}

// PrintStat writes a positive unsigned statistic as message, value and newline.
func PrintStat(out io.Writer, message string, value uint32) error {
	if value == 0 {
		return nil
	}
	var line strings.Builder
	line.WriteString(statMessagePrefix(message))
	line.WriteString(strconv.FormatUint(uint64(value), 10))
	line.WriteByte('\n')
	_, err := io.WriteString(out, line.String())
	return err
}

// PrintStatFloat writes a positive float32 statistic with six significant
// digits, matching the source stream's default precision through strconv.
func PrintStatFloat(out io.Writer, message string, value float32) error {
	if !(value > 0) {
		return nil
	}
	var line strings.Builder
	line.WriteString(statMessagePrefix(message))
	line.WriteString(strconv.FormatFloat(float64(value), 'g', 6, 32))
	line.WriteByte('\n')
	_, err := io.WriteString(out, line.String())
	return err
}
