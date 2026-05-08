package audit_test

import (
	"strings"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/audit"
)

func TestRotateSession_HappyPath(t *testing.T) {
	in := "http://user-zone-residential-sessionId-abc:secret@host:8888/p"
	got, err := audit.RotateSession(in, "sessionId", "deadbeef")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := "http://user-zone-residential-sessionId-deadbeef:secret@host:8888/p"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRotateSession_PreservesPassword(t *testing.T) {
	in := "http://user-session-old:p%40ss%2Fword@host:1"
	got, err := audit.RotateSession(in, "session", "new")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(got, "p%40ss%2Fword") {
		t.Fatalf("password not preserved: %q", got)
	}
}

func TestRotateSession_FirstOccurrenceWins(t *testing.T) {
	in := "http://customer-session-foo-zone-session-bar:p@h:1"
	got, err := audit.RotateSession(in, "session", "new")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := "http://customer-session-new-zone-session-bar:p@h:1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRotateSession_KeyNotFound(t *testing.T) {
	in := "http://user-zone-residential:p@h:1"
	_, err := audit.RotateSession(in, "session", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in %q", err.Error())
	}
}

func TestRotateSession_KeyIsLastToken(t *testing.T) {
	in := "http://user-zone-session:p@h:1"
	_, err := audit.RotateSession(in, "session", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no following value") {
		t.Fatalf("expected 'no following value' in %q", err.Error())
	}
}

func TestRotateSession_EmptyUsername(t *testing.T) {
	in := "http://host:1"
	_, err := audit.RotateSession(in, "session", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no username") {
		t.Fatalf("expected 'no username' in %q", err.Error())
	}
}

func TestRotateSession_UnparseableURL(t *testing.T) {
	_, err := audit.RotateSession("::not a url::", "session", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRotateSession_CaseSensitive(t *testing.T) {
	in := "http://user-SessionId-abc:p@h:1"
	_, err := audit.RotateSession(in, "sessionId", "new")
	if err == nil {
		t.Fatal("expected error: case-sensitive lookup should miss SessionId vs sessionId")
	}
}

func TestNewSessionValue_NoDashes(t *testing.T) {
	for i := 0; i < 50; i++ {
		v := audit.NewSessionValue()
		if strings.Contains(v, "-") {
			t.Fatalf("session value contains dash: %q", v)
		}
		if len(v) != 16 {
			t.Fatalf("len = %d want 16", len(v))
		}
	}
}

func TestNewSessionValue_Unique(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 1000; i++ {
		v := audit.NewSessionValue()
		if _, dup := seen[v]; dup {
			t.Fatalf("duplicate within 1000 samples: %q", v)
		}
		seen[v] = struct{}{}
	}
}
