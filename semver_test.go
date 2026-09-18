package semver

import (
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
