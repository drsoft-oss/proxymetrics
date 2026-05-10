package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// dimensionsCacheTTL caps the per-window result cache. Value lists rarely change
// within this interval and the chip dropdowns are noisy.
const dimensionsCacheTTL = 30 * time.Second

type dimensionsCache struct {
	mu      sync.Mutex
	entries map[string]dimensionsCacheEntry
}

type dimensionsCacheEntry struct {
	at   time.Time
	body map[string]any
}

func dimensionsHandler(s store.Store) http.HandlerFunc {
	cache := &dimensionsCache{entries: map[string]dimensionsCacheEntry{}}

	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Truncate to TTL so requests within the same TTL window share a cache key.
		// Without this the default-window's wall-clock `now` would produce a fresh
		// key every second and the cache would never hit.
		keyFrom := f.From.Truncate(dimensionsCacheTTL)
		keyTo := f.To.Truncate(dimensionsCacheTTL)
		cacheKey := keyFrom.Format(time.RFC3339) + "|" + keyTo.Format(time.RFC3339)
		cache.mu.Lock()
		entry, ok := cache.entries[cacheKey]
		cache.mu.Unlock()
		if ok && time.Since(entry.at) < dimensionsCacheTTL {
			writeJSON(w, http.StatusOK, entry.body)
			return
		}

		dims, err := s.DistinctDimensions(r.Context(), f.From, f.To)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		profiles := make([]map[string]string, 0, len(dims.Profiles))
		for _, p := range dims.Profiles {
			profiles = append(profiles, map[string]string{"id": p.ID, "name": p.Label})
		}

		body := map[string]any{
			"vendor":       nullableSlice(dims.Vendors),
			"type":         nullableSlice(dims.Types),
			"region":       nullableSlice(dims.Regions),
			"team":         nullableSlice(dims.Teams),
			"project":      nullableSlice(dims.Projects),
			"status_class": dims.StatusClasses,
			"captcha_kind": dims.CaptchaKinds,
			"profile_id":   profiles,
		}

		cache.mu.Lock()
		cache.entries[cacheKey] = dimensionsCacheEntry{at: time.Now(), body: body}
		cache.mu.Unlock()

		writeJSON(w, http.StatusOK, body)
	}
}

// nullableSlice ensures the JSON encoder emits `[]` instead of `null` for empty slices.
func nullableSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
