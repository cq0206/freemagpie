package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/freemagpie"
	"github.com/yetone/magpie/internal/settings"
)

// creditsMux is the page's API on its own, in the test's home.
func creditsMux(t *testing.T) *http.ServeMux {
	t.Helper()
	for _, n := range []string{"freemagpie-cache.json", "freemagpie.json"} {
		if err := os.Remove(filepath.Join(settings.Dir(), n)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	freemagpie.Cache.ResetForTest()
	freemagpie.ResetStatesForTest()
	mux := http.NewServeMux()
	creditsRoutes(mux)
	return mux
}

// mockCloud serves the catalog (API.md §1) at a URL the settings name.
func mockCloud(t *testing.T, fail *bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if *fail {
			rw.WriteHeader(500)
			rw.Write([]byte(`{"error":{"code":"UNAVAILABLE","message":"down"}}`))
			return
		}
		rw.Write([]byte(`{"data":[{"id":"o1","slug":"qoder-daily","product":{"slug":"qoder","name":"Qoder","category":"coding-agent"},"title":"Daily Credits","reward_kind":"native_credit","reward_label":"30 a day","reset_period":"daily","claim_method":"manual","claim_url":"https://qoder.example/c","api_compatible":false,"verification_status":"official_verified"},{"id":"o2","slug":"free-api","product":{"slug":"example","name":"Example","category":"api-provider"},"title":"Free API tier","reward_kind":"api_credit","reward_label":"1M tokens a day","reset_period":"daily","claim_method":"manual","claim_url":"https://example.example/keys","api_compatible":true,"verification_status":"official_verified","import_link":"magpie://import?name=Example&chat=https://api.example.com/v1&models=m1"}],"next_cursor":null}`))
	}))
}

// TestCreditsOffersReadAndCache: the page's route answers the cloud's
// catalog; with the cloud down it answers the cache with the error said
// beside it, and with no cache at all it says the error alone — never
// an empty catalog passed off as fresh.
func TestCreditsOffersReadAndCache(t *testing.T) {
	mux := creditsMux(t)
	fail := false
	cloud := mockCloud(t, &fail)
	defer cloud.Close()
	settings.Save(settings.Settings{FreeMagpieURL: cloud.URL})

	get := func(reload string) (int, map[string]any) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/credits/offers"+reload, nil))
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	code, out := get("")
	if code != 200 {
		t.Fatalf("read: %d %v", code, out)
	}
	if n := len(out["data"].([]any)); n != 2 {
		t.Fatalf("offers: %d", n)
	}
	if _, ok := out["error"]; ok {
		t.Fatalf("a good read carried an error: %v", out["error"])
	}

	// the cloud goes down: a Refresh asks it anyway, and the cache
	// serves the page, with its age and the error beside it (a plain
	// read answers the fresh cache without asking, as the test below
	// holds)
	fail = true
	code, out = get("?reload=1")
	if code != 200 {
		t.Fatalf("cached read: %d", code)
	}
	if n := len(out["data"].([]any)); n != 2 {
		t.Fatalf("cached offers: %d", n)
	}
	if out["from_cache"] != true {
		t.Fatal("the cached read isn't said to be one")
	}
	if e, _ := out["error"].(string); !strings.Contains(e, "down") && !strings.Contains(e, "500") && !strings.Contains(e, "UNAVAILABLE") {
		t.Fatalf("the cloud's error isn't said: %v", out["error"])
	}

	// no cache at all (a fresh home, the cloud still down): the error
	// alone, no empty data passed off as a catalog
	creditsMux(t)
	code, out = get("?reload=1")
	if code != 200 {
		t.Fatalf("no cache: %d", code)
	}
	if d, ok := out["data"].([]any); ok && len(d) > 0 {
		t.Fatalf("an error answer held offers: %v", d)
	}
	if _, ok := out["error"]; !ok {
		t.Fatal("no error said")
	}
}

// TestCreditsStateRoundTrip: the page's state route keeps saved and
// claimed, and refuses a state it doesn't know.
func TestCreditsStateRoundTrip(t *testing.T) {
	mux := creditsMux(t)
	post := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/credits/state", strings.NewReader(body)))
		return rec
	}
	if rec := post(`{"slug":"qoder-daily","state":"saved"}`); rec.Code != 200 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	if rec := post(`{"slug":"qoder-daily","state":"claimed","claimedAt":"2026-11-13T09:00:00Z"}`); rec.Code != 200 {
		t.Fatalf("claim: %d %s", rec.Code, rec.Body)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/credits/state", nil))
	var rows map[string]freemagpie.OfferState
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("read: %v (%s)", err, rec.Body)
	}
	if rows["qoder-daily"].State != "claimed" {
		t.Fatalf("rows: %+v", rows)
	}
	if rec := post(`{"slug":"x","state":"weird"}`); rec.Code != 400 {
		t.Fatalf("a state it doesn't know: %d", rec.Code)
	}
	if rec := post(`{"state":"saved"}`); rec.Code != 400 {
		t.Fatalf("no slug: %d", rec.Code)
	}
}

// TestCreditsFreshCacheServesItself: a cache read within the hour is
// answered without the cloud being asked at all (the page opens fast,
// and a cloud set to "off" still shows the catalog).
func TestCreditsFreshCacheServesItself(t *testing.T) {
	mux := creditsMux(t)
	fail := false
	cloud := mockCloud(t, &fail)
	defer cloud.Close()
	settings.Save(settings.Settings{FreeMagpieURL: cloud.URL})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/credits/offers", nil))
	if rec.Code != 200 {
		t.Fatalf("first: %d", rec.Code)
	}

	// the cloud goes off entirely: the fresh cache is still the answer
	settings.Save(settings.Settings{FreeMagpieURL: "off"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/credits/offers", nil))
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if rec.Code != 200 || len(out["data"].([]any)) != 2 || out["from_cache"] != true {
		t.Fatalf("off with a fresh cache: %d %v", rec.Code, out)
	}
	// and Reload with no cloud says the error, keeping the cache
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/credits/offers?reload=1", nil))
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if _, ok := out["error"]; !ok || len(out["data"].([]any)) != 2 {
		t.Fatalf("reload with no cloud: %v", out)
	}
}
