package core

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"encoding/base64"
	"fmt"

	"github.com/elazarl/goproxy"
	"github.com/oklog/ulid/v2"

	"github.com/drsoft-oss/proxymetrics/internal/profile"
	"github.com/drsoft-oss/proxymetrics/internal/proxy/ca"
	"github.com/drsoft-oss/proxymetrics/internal/proxy/credtags"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// EventEmitter receives finalized events from the hot path.
type EventEmitter interface {
	Submit(e store.Event) bool
}

// CostFunc returns the cost_usd to record for an event.
type CostFunc func(p store.Profile, bytesIn int64, ts time.Time) float64

// Logger is the minimal logging interface the core uses; pluggable for tests.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Broadcaster is the interface that the optional live-event fan-out implements.
type Broadcaster interface {
	Publish(e store.Event)
}

// Config bundles the core's collaborators.
type Config struct {
	DeploymentSecret string
	Registry         *profile.Registry
	LeafFactory      *ca.LeafFactory
	Emitter          EventEmitter
	Cost             CostFunc
	DefaultTeam      string
	DefaultProject   string
	Logger           Logger
	Broadcaster      Broadcaster // optional; nil-safe
}

type trackerKey struct{}

type tracker struct {
	requestID  string
	profile    store.Profile
	team       string
	project    string
	startedAt  time.Time
	statusCode int
	bytesIn    atomic.Int64
	bytesOut   atomic.Int64
	host       string
	pathHash   string
	cfg        *Config
	emitted    atomic.Bool
}

// New returns a configured *goproxy.ProxyHttpServer ready to serve.
func New(cfg Config) (*goproxy.ProxyHttpServer, error) {
	if cfg.LeafFactory == nil {
		return nil, errors.New("core: LeafFactory is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("core: Logger is required")
	}

	p := goproxy.NewProxyHttpServer()
	p.Verbose = false

	// Custom transport: routes each request through the upstream proxy URL stored
	// on the resolved profile (looked up via the tracker stashed in request context).
	p.Tr = transport()

	// CONNECT handler: enforce auth, then MITM with our leaf cert.
	p.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		if !cfg.authConnect(ctx.Req) {
			ctx.Resp = unauth("missing or wrong proxy auth")
			return goproxy.RejectConnect, host
		}
		// Build the tracker now so the inner forwarded request has it in context.
		t := newTracker(ctx.Req, &cfg)
		ctx.UserData = t
		return &goproxy.ConnectAction{
			Action:    goproxy.ConnectMitm,
			TLSConfig: cfg.tlsConfigForHost,
		}, host
	})

	// Plain HTTP request handler: enforce auth and build tracker for the request.
	p.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		// For HTTPS-tunneled requests goproxy invokes this AGAIN after MITM with
		// the inner request. ctx.UserData was set in HandleConnectFunc; reuse it.
		t, _ := ctx.UserData.(*tracker)
		if t == nil {
			// Plain HTTP path. Auth must be enforced here.
			if !cfg.authConnect(req) {
				return req, unauth("missing or wrong proxy auth")
			}
			t = newTracker(req, &cfg)
			ctx.UserData = t
		}

		// Stamp the request with target metadata.
		host := NormalizeHost(req.URL.Host)
		if host == "" {
			host = NormalizeHost(req.Host)
		}
		t.host = host
		t.pathHash = PathHash(req.URL.RequestURI())

		// Wrap request body to count upstream-bound bytes.
		if req.Body != nil && req.Body != http.NoBody {
			req.Body = &countingReadCloser{rc: req.Body, w: &t.bytesOut}
		}

		// Stash the tracker in request context so transport's Proxy func can read the
		// resolved profile.
		ctx2 := context.WithValue(req.Context(), trackerKey{}, t)
		req2 := req.WithContext(ctx2)
		return req2, nil
	})

	// Response handler: capture status code, wrap body for byte counting, emit
	// event after response body is fully streamed.
	p.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		t, _ := ctx.UserData.(*tracker)
		if t == nil {
			return resp
		}
		if resp == nil {
			// Upstream failed; ctx.Error carries the cause.
			t.emit(classifyErr(ctx.Error))
			return resp
		}
		t.statusCode = resp.StatusCode
		if resp.Body != nil {
			resp.Body = &finalizingReadCloser{
				rc:      &countingReadCloser{rc: resp.Body, w: &t.bytesIn},
				onClose: func() { t.emit(nil) },
			}
		} else {
			t.emit(nil)
		}
		return resp
	})

	// HTTPS MITM path: if the upstream connection fails after CONNECT, goproxy
	// may not invoke OnResponse at all. ConnectionErrHandler catches those cases.
	p.ConnectionErrHandler = func(w io.Writer, ctx *goproxy.ProxyCtx, err error) {
		t, _ := ctx.UserData.(*tracker)
		if t != nil {
			t.emit(classifyErr(err))
		}
		// Don't write anything to w — goproxy handles the connection state.
	}

	return p, nil
}

func (c Config) authConnect(req *http.Request) bool {
	_, pass, ok := ParseProxyAuth(req)
	if !ok {
		return false
	}
	return CheckSecret(pass, c.DeploymentSecret)
}

func (c Config) tlsConfigForHost(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	bare := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		bare = h
	}
	cert, err := c.LeafFactory.Certificate(bare)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{*cert}, MinVersion: tls.VersionTLS12}, nil
}

func transport() *http.Transport {
	return &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			t, _ := req.Context().Value(trackerKey{}).(*tracker)
			if t == nil {
				return nil, nil
			}
			return url.Parse(t.profile.UpstreamURL)
		},
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		ForceAttemptHTTP2: true,
	}
}

func newTracker(req *http.Request, cfg *Config) *tracker {
	id := ulid.Make().String()
	user, _, _ := ParseProxyAuth(req)

	upstreamURL, tags, decodeErr := decodeUpstream(user)
	if decodeErr != nil {
		cfg.Logger.Warn("could not decode upstream URL from proxy auth username; forwarding will fail",
			"request_id", id, "err", decodeErr.Error())
	}

	return &tracker{
		requestID: id,
		profile:   synthesizeProfile(tags, upstreamURL),
		team:      cfg.DefaultTeam,
		project:   cfg.DefaultProject,
		startedAt: time.Now().UTC(),
		cfg:       cfg,
	}
}

// decodeUpstream pulls the upstream proxy URL out of the proxy-auth username.
// The username is base64-encoded (raw URL or standard, with or without
// padding) and decodes to a full proxy URL like
// "http://user-with-tags:pass@host:port". From the URL's userinfo we lift the
// ProxyMetrics-reserved tags (provider/type/price) via credtags.Parse, and
// rebuild the forwarding URL with the cleaned (stripped) username.
func decodeUpstream(authUser string) (forwardURL string, tags credtags.Tags, err error) {
	if authUser == "" {
		return "", credtags.Tags{}, fmt.Errorf("empty proxy-auth username")
	}
	raw, err := decodeBase64Lenient(authUser)
	if err != nil {
		return "", credtags.Tags{}, fmt.Errorf("base64 decode: %w", err)
	}
	u, err := url.Parse(string(raw))
	if err != nil {
		return "", credtags.Tags{}, fmt.Errorf("url parse: %w", err)
	}
	if u.Host == "" {
		return "", credtags.Tags{}, fmt.Errorf("upstream URL has no host")
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "http"
	}

	var (
		upstreamUser string
		upstreamPass string
	)
	if u.User != nil {
		upstreamUser = u.User.Username()
		upstreamPass, _ = u.User.Password()
	}
	tags = credtags.Parse(upstreamUser)

	out := &url.URL{Scheme: scheme, Host: u.Host}
	if tags.Stripped != "" || upstreamPass != "" {
		out.User = url.UserPassword(tags.Stripped, upstreamPass)
	}
	return out.String(), tags, nil
}

func decodeBase64Lenient(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("not valid base64")
}

// synthesizeProfile builds an in-memory store.Profile from the parsed credtags
// and the already-built forwarding URL. The resulting profile is what the hot
// path uses for routing (UpstreamURL) and what the event writer copies onto
// each event row (ID, Vendor, Type, PricePerGB).
func synthesizeProfile(tags credtags.Tags, upstreamURL string) store.Profile {
	var price *float64
	if tags.PriceCentsPerGB > 0 {
		v := float64(tags.PriceCentsPerGB) / 100.0
		price = &v
	}
	return store.Profile{
		ID:          profile.SyntheticID(tags.Provider, tags.Type),
		Label:       profile.SyntheticLabel(tags.Provider, tags.Type),
		Vendor:      tags.Provider,
		Type:        tags.Type,
		UpstreamURL: upstreamURL,
		PricePerGB:  price,
		Currency:    "USD",
	}
}

func (t *tracker) emit(errClass error) {
	if !t.emitted.CompareAndSwap(false, true) {
		return
	}
	latency := int(time.Since(t.startedAt) / time.Millisecond)
	bytesIn := t.bytesIn.Load()
	bytesOut := t.bytesOut.Load()

	cls := StatusClass(t.statusCode, errClass)
	cost := t.cfg.Cost(t.profile, bytesIn, t.startedAt)

	event := store.Event{
		TS:             time.Now().UTC(),
		RequestID:      t.requestID,
		ProfileID:      t.profile.ID,
		Vendor:         t.profile.Vendor,
		Type:           t.profile.Type,
		Region:         t.profile.Region,
		TargetHost:     t.host,
		TargetPathHash: t.pathHash,
		StatusCode:     t.statusCode,
		StatusClass:    cls,
		BytesIn:        bytesIn,
		BytesOut:       bytesOut,
		LatencyMS:      latency,
		CostUSD:        cost,
		Team:           t.team,
		Project:        t.project,
	}
	t.cfg.Emitter.Submit(event)
	if t.cfg.Broadcaster != nil {
		t.cfg.Broadcaster.Publish(event)
	}
}

// classifyErr maps an arbitrary error from goproxy/http.Transport to the
// sentinel ErrTimeout when the underlying cause is a deadline exceeded; any
// other non-nil error becomes a generic "connection_error" sentinel.
func classifyErr(err error) error {
	if err == nil {
		return nil
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTimeout
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	return err // any non-nil error → StatusClass returns "connection_error"
}

// --- helpers ---

func unauth(msg string) *http.Response {
	body := strings.NewReader(msg + "\n")
	return &http.Response{
		StatusCode: http.StatusProxyAuthRequired,
		Status:     "407 Proxy Authentication Required",
		Header: http.Header{
			"Proxy-Authenticate": []string{`Basic realm="proxymetrics"`},
			"Content-Type":       []string{"text/plain"},
		},
		Body:          io.NopCloser(body),
		ContentLength: int64(body.Len()),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
}

// countingReadCloser wraps an io.ReadCloser updating an atomic byte counter.
type countingReadCloser struct {
	rc io.ReadCloser
	w  *atomic.Int64
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.rc.Read(p)
	if n > 0 {
		c.w.Add(int64(n))
	}
	return n, err
}

func (c *countingReadCloser) Close() error { return c.rc.Close() }

// finalizingReadCloser fires onClose exactly once when the body is closed OR
// when EOF is observed on Read (whichever comes first).
type finalizingReadCloser struct {
	rc      io.ReadCloser
	onClose func()
	fired   atomic.Bool
}

func (f *finalizingReadCloser) Read(p []byte) (int, error) {
	n, err := f.rc.Read(p)
	if errors.Is(err, io.EOF) && f.fired.CompareAndSwap(false, true) {
		f.onClose()
	}
	return n, err
}

func (f *finalizingReadCloser) Close() error {
	err := f.rc.Close()
	if f.fired.CompareAndSwap(false, true) {
		f.onClose()
	}
	return err
}
