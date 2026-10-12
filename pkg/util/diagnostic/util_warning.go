// Source: src/util/warning.h:23-53; src/util/warning.cpp:49-85,106-168.
package diagnostic

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

// WarningService is the concrete single-goroutine owner of warning.cpp's
// process-wide scalar settings. Streams are borrowed from the caller.
type WarningService struct {
	stderr  io.Writer
	stream  io.Writer
	enabled bool
}

func NewWarningService(stderr io.Writer) *WarningService {
	if stderr == nil {
		stderr = os.Stderr
	}
	return &WarningService{stderr: stderr, enabled: true}
}

func (s *WarningService) Enable(flag bool) error {
	if s == nil {
		return arena.ErrConfig
	}
	s.enabled = flag
	return nil
}
func (s *WarningService) Enabled() bool { return s != nil && s.enabled }

func (s *WarningService) SetWarningStream(stream io.Writer) error {
	if s == nil {
		return arena.ErrConfig
	}
	s.stream = stream
	return nil
}

// Warningf covers retained warning_msg calls using Go's %s and %d formatting.
// String arguments are C-string views at those source sites.
func (s *WarningService) Warningf(format string, args ...any) error {
	if s == nil {
		return arena.ErrConfig
	}
	if !s.enabled {
		return nil
	}
	if i := strings.IndexByte(format, 0); i >= 0 {
		format = format[:i]
	}
	copied := false
	for i, arg := range args {
		if value, ok := arg.(string); ok {
			if end := strings.IndexByte(value, 0); end >= 0 {
				if !copied {
					args = append([]any(nil), args...)
					copied = true
				}
				args[i] = value[:end]
			}
		}
	}
	return s.Warning(fmt.Sprintf(format, args...))
}

// Warning writes one literal C-string message. On the selected Linux AMD64
// source profile, the optional ostream route truncates formatted content to
// 1023 bytes; the default FILE route does not truncate.
func (s *WarningService) Warning(message string) error {
	if s == nil {
		return arena.ErrConfig
	}
	if !s.enabled {
		return nil
	}
	if i := strings.IndexByte(message, 0); i >= 0 {
		message = message[:i]
	}
	w := s.stream
	if w != nil {
		if len(message) > 1023 {
			message = message[:1023]
		}
	} else {
		w = s.stderr
	}
	if _, e := io.WriteString(w, "WARNING: "); e != nil {
		return e
	}
	if _, e := io.WriteString(w, message); e != nil {
		return e
	}
	_, e := io.WriteString(w, "\n")
	return e
}
