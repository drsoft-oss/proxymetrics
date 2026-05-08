package profile

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/ipcheck"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// ProxyClientBuilder produces an *http.Client whose outbound requests are routed
// through the given profile's upstream proxy. The serve command wires the real
// implementation; tests can stub it.
type ProxyClientBuilder interface {
	ClientFor(p store.Profile) (*http.Client, error)
}

// DefaultProxyClientBuilder builds an http.Client whose Transport.Proxy returns
// the profile's UpstreamURL.
type DefaultProxyClientBuilder struct {
	Timeout time.Duration
}

func (d DefaultProxyClientBuilder) ClientFor(p store.Profile) (*http.Client, error) {
	u, err := url.Parse(p.UpstreamURL)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{Proxy: http.ProxyURL(u)}
	t := d.Timeout
	if t == 0 {
		t = 15 * time.Second
	}
	return &http.Client{Transport: tr, Timeout: t}, nil
}

// RegisterTestEndpoint wires the /test suffix handler. The mux parameter is kept
// for signature consistency with serve.go but is not used — the route is
// dispatched by the catch-all handler in RegisterHTTP.
func RegisterTestEndpoint(_ *http.ServeMux, r *Registry, ipc *ipcheck.Client, builder ProxyClientBuilder) {
	testEndpointHandler = func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.NotFound(w, req)
			return
		}
		path := strings.TrimPrefix(req.URL.Path, "/api/v1/profiles/")
		id := strings.TrimSuffix(path, "/test")
		p, ok := r.Lookup(id)
		if !ok {
			writeProblem(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		client, err := builder.ClientFor(p)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "client_build", err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		ipc2 := ipc.WithClient(client)

		start := time.Now()
		res, err := ipc2.Check(ctx)
		latency := int(time.Since(start) / time.Millisecond)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"exit_ip":     "",
				"latency_ms":  latency,
				"status_code": 0,
				"api_used":    "",
				"error":       err.Error(),
			})
			return
		}

		out := map[string]any{
			"exit_ip":     res.IP,
			"latency_ms":  latency,
			"status_code": 200,
			"api_used":    res.APIUsed,
		}
		if p.PricePerGB == nil {
			out["hint"] = "price_per_gb is unset; cost tracking is disabled for this profile"
		}
		writeJSON(w, http.StatusOK, out)
	}
}
