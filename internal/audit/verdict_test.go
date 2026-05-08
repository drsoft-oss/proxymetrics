package audit_test

import (
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/audit"
	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

func TestEvaluate_LocationCountryMatch(t *testing.T) {
	v := audit.Evaluate(audit.Expected{
		Country: "RO", CheckLevel: "country", Type: "residential",
	}, geo.Result{Country: "RO", IsDatacenter: false, IsMobile: false})
	if !v.LocationMatch || !v.TypeMatch {
		t.Fatalf("got %+v", v)
	}
}

func TestEvaluate_LocationCountryMismatch(t *testing.T) {
	v := audit.Evaluate(audit.Expected{
		Country: "RO", CheckLevel: "country", Type: "residential",
	}, geo.Result{Country: "DE"})
	if v.LocationMatch {
		t.Fatalf("expected mismatch, got %+v", v)
	}
}

func TestEvaluate_StateNormalization(t *testing.T) {
	v := audit.Evaluate(audit.Expected{
		Country: "FR", State: "Île-de-France", CheckLevel: "state", Type: "residential",
	}, geo.Result{Country: "FR", State: "Ile-de-france"})
	if !v.LocationMatch {
		t.Fatalf("expected match after diacritics strip, got %+v", v)
	}
}

func TestEvaluate_CityWithin50km(t *testing.T) {
	// Bucharest centroid 44.43, 26.10 — 25 km offset
	v := audit.Evaluate(audit.Expected{
		Country: "RO", City: "Bucharest", Lat: 44.43, Lon: 26.10,
		CheckLevel: "city", Type: "residential",
	}, geo.Result{Country: "RO", Lat: 44.65, Lon: 26.10})
	if !v.LocationMatch {
		t.Fatalf("expected match within 50 km, got %+v", v)
	}
}

func TestEvaluate_CityBeyond50km(t *testing.T) {
	// 200 km offset
	v := audit.Evaluate(audit.Expected{
		Country: "RO", City: "Bucharest", Lat: 44.43, Lon: 26.10,
		CheckLevel: "city", Type: "residential",
	}, geo.Result{Country: "RO", Lat: 46.20, Lon: 26.10})
	if v.LocationMatch {
		t.Fatalf("expected mismatch beyond 50 km, got %+v", v)
	}
}

func TestEvaluate_TypeResidentialDatacenter(t *testing.T) {
	v := audit.Evaluate(audit.Expected{
		Country: "RO", CheckLevel: "country", Type: "residential",
	}, geo.Result{Country: "RO", IsDatacenter: true})
	if v.TypeMatch {
		t.Fatalf("expected type mismatch, got %+v", v)
	}
}

func TestEvaluate_TypeResidentialMobile(t *testing.T) {
	v := audit.Evaluate(audit.Expected{
		Country: "RO", CheckLevel: "country", Type: "residential",
	}, geo.Result{Country: "RO", IsMobile: true})
	if v.TypeMatch {
		t.Fatalf("expected type mismatch (residential vs mobile), got %+v", v)
	}
}

func TestEvaluate_TypeMobile(t *testing.T) {
	good := audit.Evaluate(audit.Expected{Country: "RO", CheckLevel: "country", Type: "mobile"},
		geo.Result{Country: "RO", IsMobile: true})
	if !good.TypeMatch {
		t.Fatalf("expected mobile match, got %+v", good)
	}
	bad := audit.Evaluate(audit.Expected{Country: "RO", CheckLevel: "country", Type: "mobile"},
		geo.Result{Country: "RO", IsMobile: false})
	if bad.TypeMatch {
		t.Fatalf("expected mismatch when not mobile, got %+v", bad)
	}
}

func TestEvaluate_TypeDatacenter(t *testing.T) {
	good := audit.Evaluate(audit.Expected{Country: "RO", CheckLevel: "country", Type: "datacenter"},
		geo.Result{Country: "RO", IsDatacenter: true})
	if !good.TypeMatch {
		t.Fatalf("expected datacenter match, got %+v", good)
	}
	bad := audit.Evaluate(audit.Expected{Country: "RO", CheckLevel: "country", Type: "datacenter"},
		geo.Result{Country: "RO", IsDatacenter: false})
	if bad.TypeMatch {
		t.Fatalf("expected mismatch when not datacenter, got %+v", bad)
	}
}
