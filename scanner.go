package semver

import (
	"bufio"
	"io"
	"strings"
)

// Result is one line's worth of scanning: either a parsed Version or the
// error explaining why that line was not one.
type Result struct {
	Line    int
	Raw     string
	Version Version
	Err     error
}

// Scanner validates a stream of newline-separated version strings without
// ever holding more than one line in memory at a time. That matters
// because the intended input is things like a full git tag dump or a
// package registry export, which can run to gigabytes.
type Scanner struct {
	r       *bufio.Reader
	lineNum int
	result  Result
	ioErr   error
	done    bool
}

// NewScanner wraps r for line-by-line version validation.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

// Next advances to the next non-blank line and reports whether one was
// found. On false, callers should check Err to distinguish a clean EOF
// from an I/O failure.
func (s *Scanner) Next() bool {
	if s.done {
		return false
	}
	for {
		line, err := s.r.ReadString('\n')
		if err != nil && err != io.EOF {
			s.ioErr = err
			s.done = true
			return false
		}
		atEOF := err == io.EOF
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if atEOF {
				s.done = true
				return false
			}
			continue
		}

		s.lineNum++
		v, perr := Parse(line)
		s.result = Result{Line: s.lineNum, Raw: line, Version: v, Err: perr}
		if atEOF {
			// Yield this last line now; the next call reports done.
			s.done = true
		}
		return true
	}
}

// Result returns the line most recently produced by Next.
func (s *Scanner) Result() Result {
	return s.result
}

// Err returns the first I/O error encountered, if any. It does not report
// per-line validation failures; those are carried on each Result.
func (s *Scanner) Err() error {
	return s.ioErr
}
