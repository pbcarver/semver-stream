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

// ParseLoose is like Parse but accepts a single leading "v" or "V", as
// found in most git tags. Nothing else is relaxed: "vv1.2.3", "v 1.2.3"
// and a bare "v" are still errors. The prefix is not kept, so String on
// the result gives the canonical form without it.
func ParseLoose(s string) (Version, error) {
	if s != "" && (s[0] == 'v' || s[0] == 'V') {
		s = s[1:]
	}
	return Parse(s)
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

// Compare returns -1, 0, or 1 as v is less than, equal to, or greater
// than other, following semver's precedence rules: core version numbers
// compare numerically, a prerelease version has lower precedence than
// the same version without one, and prerelease identifiers compare
// left to right (numeric identifiers numerically, alphanumeric ones
// ASCII-lexically, with a shorter list preceding a longer one when it
// is otherwise a prefix). Build metadata is ignored, matching the spec.
func Compare(v, other Version) int {
	if c := compareUint(v.Major, other.Major); c != 0 {
		return c
	}
	if c := compareUint(v.Minor, other.Minor); c != 0 {
		return c
	}
	if c := compareUint(v.Patch, other.Patch); c != 0 {
		return c
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func compareUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// comparePrerelease implements semver.org spec item 11: a version with a
// prerelease section always has lower precedence than one without.
func comparePrerelease(a, b []string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1
	}
	if len(b) == 0 {
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := compareIdentifier(a[i], b[i]); c != 0 {
			return c
		}
	}
	return compareUint(uint64(len(a)), uint64(len(b)))
}

// compareIdentifier compares a single prerelease identifier pair. Both
// were already validated by Parse, so isDigits alone is enough to tell
// numeric from alphanumeric here.
func compareIdentifier(a, b string) int {
	aIsNum, bIsNum := isDigits(a), isDigits(b)
	if aIsNum && bIsNum {
		an, _ := strconv.ParseUint(a, 10, 64)
		bn, _ := strconv.ParseUint(b, 10, 64)
		return compareUint(an, bn)
	}
	if aIsNum != bIsNum {
		// Numeric identifiers always have lower precedence than
		// alphanumeric ones (spec item 11.4.3).
		if aIsNum {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// Versions is a slice of Version that implements sort.Interface in
// semver precedence order.
type Versions []Version

func (vs Versions) Len() int           { return len(vs) }
func (vs Versions) Less(i, j int) bool { return Compare(vs[i], vs[j]) < 0 }
func (vs Versions) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }

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
