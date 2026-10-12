// Source: src/util/warning.h:23-53; src/util/warning.cpp:49-85,106-168.
package diagnostic

import (
	"io"
	"os"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

// WarningService is the concrete single-goroutine owner of warning.cpp's
// process-wide scalar settings. Streams are borrowed from the caller.
type WarningService struct {
	stdout, stderr io.Writer
	stream         io.Writer
	enabled        bool
	toStdout       bool
}

func NewWarningService(stdout, stderr io.Writer) *WarningService {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &WarningService{stdout: stdout, stderr: stderr, enabled: true}
}

func (s *WarningService) Enable(flag bool) error {
	if s == nil {
		return arena.ErrConfig
	}
	s.enabled = flag
	return nil
}
func (s *WarningService) Enabled() bool { return s != nil && s.enabled }

func (s *WarningService) SendToStdout(flag bool) error {
	if s == nil {
		return arena.ErrConfig
	}
	s.toStdout = flag
	return nil
}
func (s *WarningService) SetWarningStream(stream io.Writer) error {
	if s == nil {
		return arena.ErrConfig
	}
	s.stream = stream
	return nil
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
	} else if s.toStdout {
		w = s.stdout
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

// WarningDisplayer preserves warning.h's sign-once/reset behavior. A disabled
// warning still counts as displayed, as in the source sign() method.
type WarningDisplayer struct {
	service   *WarningService
	message   string
	displayed bool
}

func NewWarningDisplayer(service *WarningService, message string) (*WarningDisplayer, error) {
	if service == nil {
		return nil, arena.ErrConfig
	}
	return &WarningDisplayer{service: service, message: message}, nil
}
func (d *WarningDisplayer) Sign() error {
	if d == nil || d.service == nil {
		return arena.ErrConfig
	}
	if d.displayed {
		return nil
	}
	if e := d.service.Warning(d.message); e != nil {
		return e
	}
	d.displayed = true
	return nil
}
func (d *WarningDisplayer) Reset() {
	if d != nil {
		d.displayed = false
	}
}
