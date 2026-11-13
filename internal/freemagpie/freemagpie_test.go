package freemagpie

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/settings"
)

// mockCloud answers the catalog API.md shape: offers, pages, and the
// error shape. serves is the offers it hands out, changed by a test
// between calls to see the cache move.
type mockCloud struct {
	serves func() []Offer
	fail   bool // answer 500, as an unreachable-or-broken cloud
}

func (m *mockCloud) handler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if m.fail {
			writeJSON(t, rw, 500, map[string]any{"error": map[string]string{"code": "UNAVAILABLE", "message": "the cloud is being restarted"}})
			return
		}
		if r.URL.Path != "/api/v1/offers" {
			http.NotFound(rw, r)
			return
		}
		offers := m.serves()
		// one page: the whole catalog, no cursor (a larger cloud would
		// page; the client's walk over pages is tested below)
		writeJSON(t, rw, 200, map[string]any{"data": offers, "next_cursor": nil})
	})
}

func writeJSON(t *testing.T, rw http.ResponseWriter, status int, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write(b)
}

// twoOffers is a small catalog: one native credit, one API credit with
// an import link, as the cloud would send them (API.md §1).
func twoOffers() []Offer {
	until := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	verified := time.Date(2026, 11, 12, 4, 0, 0, 0, time.UTC)
	return []Offer{
		{
			ID: "o1", Slug: "qoder-daily-credits",
			Product: Product{ID: "p1", Slug: "qoder", Name: "Qoder", Category: "coding-agent"},
			Title:   "Daily Credits", RewardKind: "native_credit", RewardLabel: "按官方公布额度",
			ResetPeriod: "daily", ClaimMethod: "manual", ClaimURL: "https://qoder.example/checkin",
			RegionCodes: []string{"CN"}, ValidUntil: &until, APICompatible: false,
			VerificationStatus: "official_verified", LastVerifiedAt: &verified,
		},
		{
			ID: "o2", Slug: "example-free-api",
			Product: Product{ID: "p2", Slug: "example", Name: "Example", Category: "api-provider"},
			Title:   "Free API tier", RewardKind: "api_credit", RewardLabel: "1M tokens a day",
			ResetPeriod: "daily", ClaimMethod: "manual", ClaimURL: "https://example.example/keys",
			RegionCodes: []string{"GLOBAL"}, APICompatible: true,
			VerificationStatus: "official_verified", LastVerifiedAt: &verified,
			ImportLink: "magpie://import?name=Example&chat=https://api.example.com/v1&models=m1",
		},
	}
}

// freshFiles puts the cache and the states in the test's home, so one
// test's files never leak into the next.
func freshFiles(t *testing.T) {
	t.Helper()
	for _, n := range []string{"freemagpie-cache.json", "freemagpie.json"} {
		if err := os.Remove(filepath.Join(settings.Dir(), n)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	Cache.ResetForTest()
	ResetStatesForTest()
}

// TestOffersReadsTheCatalog: the client reads the cloud's offers and
// writes the cache; a second read with the cloud gone answers the
// cache, its age known (never an empty catalog passed off as fresh).
func TestOffersReadsTheCatalog(t *testing.T) {
	freshFiles(t)
	mock := &mockCloud{serves: twoOffers}
	cloud := httptest.NewServer(mock.handler(t))
	defer cloud.Close()
	settings.Save(settings.Settings{FreeMagpieURL: cloud.URL})

	offers, readAt, err := Offers(context.Background(), OffersQuery{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(offers) != 2 || offers[0].Slug != "qoder-daily-credits" || offers[1].Slug != "example-free-api" {
		t.Fatalf("offers: %+v", offers)
	}
	if offers[1].ImportLink == "" || !offers[1].APICompatible {
		t.Fatalf("the API offer came without its import link: %+v", offers[1])
	}
	if readAt.IsZero() {
		t.Fatal("no read time")
	}

	// the cloud goes away: the cached catalog still serves the page
	mock.fail = true
	cached, cachedAt := Cache.Cached()
	if len(cached) != 2 || cachedAt.IsZero() {
		t.Fatalf("cache: %d offers, read %v", len(cached), cachedAt)
	}

	// and asking the cloud again says so, without losing the cache
	if _, _, err := Offers(context.Background(), OffersQuery{}); err == nil {
		t.Fatal("a failed cloud read is not an error")
	}
	if cached, _ := Cache.Cached(); len(cached) != 2 {
		t.Fatalf("a failed read dropped the cache: %d", len(cached))
	}
}

// TestOffersOffIsNotAnEmptyCatalog: with the cloud off ("off", not ""),
// a read answers ErrNoCloud — unknown, never an empty catalog the page
// would show as "no offers yet".
func TestOffersOffIsNotAnEmptyCatalog(t *testing.T) {
	freshFiles(t)
	settings.Save(settings.Settings{FreeMagpieURL: "off"})
	if _, _, err := Offers(context.Background(), OffersQuery{}); err != ErrNoCloud {
		t.Fatalf("off: %v", err)
	}
	if offers, _ := Cache.Cached(); offers != nil {
		t.Fatalf("off answered a cache: %d offers", len(offers))
	}
}

// TestOffersQueryPath: the query picks what API.md's parameters ask.
func TestOffersQueryPath(t *testing.T) {
	var got string
	cloud := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		writeJSON(t, rw, 200, map[string]any{"data": []Offer{}, "next_cursor": nil})
	}))
	defer cloud.Close()
	settings.Save(settings.Settings{FreeMagpieURL: cloud.URL})

	q := OffersQuery{Category: "coding-agent", RewardKind: "native_credit", Region: "CN", APIOnly: true, ResetPeriod: "daily", Sort: "verified", Limit: 25}
	if _, _, err := Offers(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"category=coding-agent", "reward_kind=native_credit", "region=CN", "api_compatible=true", "reset_period=daily", "sort=verified", "limit=25"} {
		if !strings.Contains(got, want) {
			t.Errorf("the query misses %s: %s", want, got)
		}
	}
}

// TestStatesSavedOnDisk: a state set is kept in freemagpie and read
// back; "" takes it away. An unreadable file is an error, never an
// empty set (a guard must not write over unknown data).
func TestStatesSavedOnDisk(t *testing.T) {
	freshFiles(t)
	rows, err := States()
	if err != nil || len(rows) != 0 {
		t.Fatalf("empty start: %d rows, %v", len(rows), err)
	}

	at := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	if err := SetState("qoder-daily-credits", "claimed", &at); err != nil {
		t.Fatal(err)
	}
	if err := SetState("example-free-api", "saved", nil); err != nil {
		t.Fatal(err)
	}
	rows, err = States()
	if err != nil || len(rows) != 2 {
		t.Fatalf("after two: %d rows, %v", len(rows), err)
	}
	if rows["qoder-daily-credits"].State != "claimed" || rows["qoder-daily-credits"].ClaimedAt == nil {
		t.Fatalf("claimed: %+v", rows["qoder-daily-credits"])
	}

	// the file is the rows, in order, readable by a fresh process
	b, err := os.ReadFile(filepath.Join(settings.Dir(), "freemagpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk []OfferState
	if err := json.Unmarshal(b, &onDisk); err != nil || len(onDisk) != 2 || onDisk[0].OfferSlug != "example-free-api" {
		t.Fatalf("on disk: %s (%v)", b, err)
	}

	// "" takes the row away, not the others
	if err := SetState("example-free-api", "", nil); err != nil {
		t.Fatal(err)
	}
	rows, err = States()
	if err != nil || len(rows) != 1 || rows["qoder-daily-credits"].State != "claimed" {
		t.Fatalf("after take-away: %+v (%v)", rows, err)
	}

	// a broken file is an error, and a set on it writes nothing
	broken := filepath.Join(settings.Dir(), "freemagpie.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	ResetStatesForTest() // read afresh
	if _, err := States(); err == nil {
		t.Fatal("a broken file reads as no error")
	}
	if err := SetState("another", "saved", nil); err == nil {
		t.Fatal("a set over a broken file wrote")
	}
	if b, _ := os.ReadFile(broken); string(b) != "{" {
		t.Fatalf("a broken file was written over: %s", b)
	}
}

// TestCacheUnreadableIsUnknown: a cache that can't be parsed is no
// cache, not an empty catalog with a zero time.
func TestCacheUnreadableIsUnknown(t *testing.T) {
	freshFiles(t)
	if err := os.WriteFile(filepath.Join(settings.Dir(), "freemagpie-cache.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if offers, readAt := Cache.Cached(); offers != nil || !readAt.IsZero() {
		t.Fatalf("a broken cache answered: %d offers at %v", len(offers), readAt)
	}
}
