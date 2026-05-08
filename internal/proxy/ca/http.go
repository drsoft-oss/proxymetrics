package ca

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
)

// HTTPHandler returns a handler exposing /cacert, /cacert.crt, /cacert/fingerprint.
func HTTPHandler(c *CA) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/cacert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/x-pem-file")
		_ = pem.Encode(w, &pem.Block{Type: "CERTIFICATE", Bytes: c.CertDER})
	})
	mux.HandleFunc("/cacert.crt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		_, _ = w.Write(c.CertDER)
	})
	mux.HandleFunc("/cacert/fingerprint", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"sha256":      c.Fingerprint(),
			"valid_until": c.Cert.NotAfter.UTC().Format("2006-01-02T15:04:05Z"),
		})
	})
	return mux
}
