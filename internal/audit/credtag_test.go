package audit_test

import (
	"reflect"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/audit"
)

func TestParseCredtag_AllFields(t *testing.T) {
	in := "http://user-provider-acme-type-residential-country-RO-state-Bucharest-city-Bucharest:secret@host.example:8080"
	got, err := audit.ParseCredtag(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := audit.Credtag{
		Provider: "acme",
		Type:     "residential",
		Country:  "RO",
		State:    "Bucharest",
		City:     "Bucharest",
		Password: "secret",
		Host:     "host.example",
		Port:     "8080",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestParseCredtag_CountryOnly(t *testing.T) {
	got, err := audit.ParseCredtag("http://user-country-US:p@h:1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Country != "US" || got.State != "" || got.City != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseCredtag_NoCredtags(t *testing.T) {
	_, err := audit.ParseCredtag("http://plainuser:p@h:1")
	if err == nil {
		t.Fatal("expected error when username has no key-value pairs")
	}
}

func TestParseCredtag_Malformed(t *testing.T) {
	_, err := audit.ParseCredtag("not a url")
	if err == nil {
		t.Fatal("expected error on malformed url")
	}
}

func TestParseCredtag_HasLevels(t *testing.T) {
	c, err := audit.ParseCredtag("http://user-country-RO-state-Bucharest:p@h:1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !c.HasLevel("country") || !c.HasLevel("state") || c.HasLevel("city") {
		t.Fatalf("levels: %+v", c)
	}
}
