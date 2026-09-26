package semver

import (
	"sort"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := []string{
		"0.0.0",
		"1.2.3",
		"10.20.30",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-0.3.7",
		"1.0.0-x.7.z.92",
		"1.0.0+20130313144700",
		"1.0.0-beta+exp.sha.5114f85",
		"2.0.0+build.1848",
	}
	for _, in := range cases {
		v, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", in, err)
			continue
		}
		if got := v.String(); got != in {
			t.Errorf("Parse(%q).String() = %q, want %q", in, got, in)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []string{
		"",
		"1",
		"1.2",
		"1.2.3.4",
		"01.2.3",
		"1.02.3",
		"1.2.03",
		"1.2.3-",
		"1.2.3-01",
		"1.2.3+",
		"1.2.3-alpha_beta!",
		"v1.2.3",
		" 1.2.3",
	}
	for _, in := range cases {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", in)
		}
	}
}

func TestCompare(t *testing.T) {
	// Ascending precedence order, taken from the semver.org spec's own
	// example plus a few extras for build metadata and equal cases.
	ordered := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
		"1.0.1",
		"1.1.0",
		"2.0.0",
	}
	for i := 0; i < len(ordered); i++ {
		vi, err := Parse(ordered[i])
		if err != nil {
			t.Fatalf("Parse(%q): %v", ordered[i], err)
		}
		if c := Compare(vi, vi); c != 0 {
			t.Errorf("Compare(%q, %q) = %d, want 0", ordered[i], ordered[i], c)
		}
		for j := i + 1; j < len(ordered); j++ {
			vj, err := Parse(ordered[j])
			if err != nil {
				t.Fatalf("Parse(%q): %v", ordered[j], err)
			}
			if c := Compare(vi, vj); c >= 0 {
				t.Errorf("Compare(%q, %q) = %d, want < 0", ordered[i], ordered[j], c)
			}
			if c := Compare(vj, vi); c <= 0 {
				t.Errorf("Compare(%q, %q) = %d, want > 0", ordered[j], ordered[i], c)
			}
		}
	}
}

func TestCompareIgnoresBuild(t *testing.T) {
	a, _ := Parse("1.0.0+build.1")
	b, _ := Parse("1.0.0+build.2")
	if c := Compare(a, b); c != 0 {
		t.Errorf("Compare with differing build metadata = %d, want 0", c)
	}
}

func TestVersionsSort(t *testing.T) {
	in := []string{"1.0.1", "1.0.0-rc.1", "2.0.0", "1.0.0-beta.11", "1.0.0"}
	want := []string{"1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "2.0.0"}

	vs := make(Versions, len(in))
	for i, s := range in {
		v, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		vs[i] = v
	}

	sort.Sort(vs)

	for i, v := range vs {
		if got := v.String(); got != want[i] {
			t.Errorf("sorted[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestScannerStream(t *testing.T) {
	input := "1.2.3\n\nnot-a-version\n1.0.0-rc.1+build.9\n"
	sc := NewScanner(strings.NewReader(input))

	var results []Result
	for sc.Next() {
		results = append(results, sc.Result())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("unexpected scanner error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0].Version.String() != "1.2.3" {
		t.Errorf("results[0] = %q, want 1.2.3", results[0].Version.String())
	}
	if results[1].Err == nil {
		t.Errorf("results[1] should have failed to parse %q", results[1].Raw)
	}
	if results[2].Version.String() != "1.0.0-rc.1+build.9" {
		t.Errorf("results[2] = %q, want 1.0.0-rc.1+build.9", results[2].Version.String())
	}
}
