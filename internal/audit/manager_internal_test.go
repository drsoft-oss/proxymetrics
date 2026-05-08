package audit

import "testing"

func TestMaskPassword_KeepsLiteralAsterisks(t *testing.T) {
	got := maskPassword("http://user-country-RO:secret@host:1/path?q=1")
	want := "http://user-country-RO:***@host:1/path?q=1"
	if got != want {
		t.Fatalf("maskPassword:\n got: %q\nwant: %q", got, want)
	}
}

func TestMaskPassword_NoUserinfoUnchanged(t *testing.T) {
	in := "http://host:1/path"
	if got := maskPassword(in); got != in {
		t.Fatalf("maskPassword(%q) = %q, want unchanged", in, got)
	}
}

func TestMaskPassword_NoPasswordUnchanged(t *testing.T) {
	in := "http://onlyuser@host:1/path"
	if got := maskPassword(in); got != in {
		t.Fatalf("maskPassword(%q) = %q, want unchanged", in, got)
	}
}
