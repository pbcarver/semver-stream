// Package semver parses and prints version strings following the
// Semantic Versioning 2.0.0 grammar (see semver.org).
package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidVersion is wrapped by every error Parse returns, so callers
// can test for it with errors.Is regardless of the specific reason.
var ErrInvalidVersion = errors.New("invalid semantic version")

// Version is a parsed semantic version. The zero value represents "0.0.0".
type Version struct {
	Major, Minor, Patch uint64
	Prerelease          []string
	Build               []string
}

// Parse validates s against the semver grammar and returns the parsed
// version. It is intentionally strict: no leading "v", no surrounding
// whitespace, no empty identifiers. Loosening any of that is a
// deliberate future decision, not an oversight.
func Parse(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("%w: empty string", ErrInvalidVersion)
	}

	main := s
	var buildPart string
	hasBuild := false
	if i := strings.IndexByte(s, '+'); i >= 0 {
		main, buildPart = s[:i], s[i+1:]
		hasBuild = true
	}

	core := main
	var prePart string
	hasPre := false
	// The '-' that introduces the prerelease section is the first one in
	// the core+prerelease string, since numeric identifiers never contain
	// a hyphen.
	if i := strings.IndexByte(main, '-'); i >= 0 {
		core, prePart = main[:i], main[i+1:]
		hasPre = true
	}

	fields := strings.Split(core, ".")
	if len(fields) != 3 {
		return Version{}, fmt.Errorf("%w: core version %q must be MAJOR.MINOR.PATCH", ErrInvalidVersion, core)
	}

	names := [3]string{"major", "minor", "patch"}
	nums := [3]uint64{}
	for i, f := range fields {
		if !isNumericIdentifier(f) {
			return Version{}, fmt.Errorf("%w: %s %q is not a valid numeric identifier", ErrInvalidVersion, names[i], f)
		}
		n, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %s %q does not fit in 64 bits", ErrInvalidVersion, names[i], f)
		}
		nums[i] = n
	}

	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}

	if hasPre {
		if prePart == "" {
			return Version{}, fmt.Errorf("%w: empty prerelease section", ErrInvalidVersion)
		}
		for _, id := range strings.Split(prePart, ".") {
			if !isAlnumHyphen(id) {
				return Version{}, fmt.Errorf("%w: prerelease identifier %q contains invalid characters", ErrInvalidVersion, id)
			}
			if isDigits(id) && !isNumericIdentifier(id) {
				return Version{}, fmt.Errorf("%w: prerelease identifier %q has a leading zero", ErrInvalidVersion, id)
			}
			v.Prerelease = append(v.Prerelease, id)
		}
	}

	if hasBuild {
		if buildPart == "" {
			return Version{}, fmt.Errorf("%w: empty build metadata section", ErrInvalidVersion)
		}
		for _, id := range strings.Split(buildPart, ".") {
			if !isAlnumHyphen(id) {
				return Version{}, fmt.Errorf("%w: build identifier %q contains invalid characters", ErrInvalidVersion, id)
			}
			v.Build = append(v.Build, id)
		}
	}

	return v, nil
}

// Validate reports whether s is a well-formed semantic version, without
// handing back the parsed value.
func Validate(s string) error {
	_, err := Parse(s)
	return err
}

// String renders v in canonical semver form.
func (v Version) String() string {
	var b strings.Builder
	b.WriteString(strconv.FormatUint(v.Major, 10))
	b.WriteByte('.')
	b.WriteString(strconv.FormatUint(v.Minor, 10))
	b.WriteByte('.')
	b.WriteString(strconv.FormatUint(v.Patch, 10))
	if len(v.Prerelease) > 0 {
		b.WriteByte('-')
		b.WriteString(strings.Join(v.Prerelease, "."))
	}
	if len(v.Build) > 0 {
		b.WriteByte('+')
		b.WriteString(strings.Join(v.Build, "."))
	}
	return b.String()
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isNumericIdentifier reports whether s is digits-only with no leading
// zero, unless s is exactly "0".
func isNumericIdentifier(s string) bool {
	if !isDigits(s) {
		return false
	}
	return len(s) == 1 || s[0] != '0'
}

// isAlnumHyphen reports whether s is a non-empty run of ASCII letters,
// digits, and hyphens, as required for prerelease and build identifiers.
func isAlnumHyphen(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c == '-':
		default:
			return false
		}
	}
	return true
}
