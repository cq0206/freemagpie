// freemagpie // The Free Credits Market page's API: the catalog as the
// freemagpie client reads it (a page-sized answer: the client walks the
// cloud's pages and caches the whole), the user's offer states, and the
// cache for when the cloud is off or unreachable. The page asks
// {reload: true} for Refresh, else this answers the cache when it is
// fresh enough and asks the cloud only when it isn't or none is read.
package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yetone/magpie/internal/freemagpie"
)

// catalogFresh is how old a cached catalog may be to be answered
// without asking the cloud: an hour, as a page of offers changes
// about that often, and the reader can always press Refresh.
const catalogFresh = time.Hour

// creditsRoutes serves the Free Credits Market page (plugins/credits):
// GET /api/credits/offers, GET /api/credits/state and
// POST /api/credits/state. It is a thin wrapper on internal/freemagpie,
// which holds the cloud client and the cache; nothing here touches the
// gateway (ARCHITECTURE.md §1.2).
func creditsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/credits/offers", func(rw http.ResponseWriter, r *http.Request) {
		reload := r.URL.Query().Get("reload") == "1"
		if !reload {
			if offers, readAt := freemagpie.Cache.Cached(); offers != nil && time.Since(readAt) < catalogFresh {
				writeJSON(rw, offersJSON{Data: offers, ReadAt: readAt, FromCache: true})
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		offers, readAt, err := freemagpie.Offers(ctx, freemagpie.OffersQuery{})
		if err != nil {
			// the cloud was asked and didn't answer: the cache still
			// serves the page, with its age and the error beside it;
			// neither is an empty catalog passed off as fresh
			if cached, cachedAt := freemagpie.Cache.Cached(); cached != nil {
				writeJSON(rw, offersJSON{Data: cached, ReadAt: cachedAt, FromCache: true, Error: err.Error()})
				return
			}
			writeJSON(rw, offersJSON{Error: err.Error()})
			return
		}
		writeJSON(rw, offersJSON{Data: offers, ReadAt: readAt})
	})
	mux.HandleFunc("GET /api/credits/state", func(rw http.ResponseWriter, r *http.Request) {
		rows, err := freemagpie.States()
		if err != nil {
			fail(rw, err)
			return
		}
		writeJSON(rw, rows)
	})
	mux.HandleFunc("POST /api/credits/state", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			Slug      string     `json:"slug"`
			State     string     `json:"state"` // saved | claimed | dismissed, "" to take it away
			ClaimedAt *time.Time `json:"claimedAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		switch in.State {
		case "", "saved", "claimed", "dismissed":
		default:
			http.Error(rw, "the state is saved, claimed, dismissed or empty", http.StatusBadRequest)
			return
		}
		if in.Slug == "" {
			http.Error(rw, "the offer's slug is missing", http.StatusBadRequest)
			return
		}
		if err := freemagpie.SetState(in.Slug, in.State, in.ClaimedAt); err != nil {
			fail(rw, err)
			return
		}
		rows, err := freemagpie.States()
		if err != nil {
			fail(rw, err)
			return
		}
		writeJSON(rw, rows)
	})
}

// offersJSON is the catalog answer: the offers, when they were read,
// whether that was the cache, and the error when the cloud was asked
// and didn't answer (the page says so beside its cached list).
type offersJSON struct {
	Data      []freemagpie.Offer `json:"data"`
	ReadAt    time.Time          `json:"read_at"`
	FromCache bool               `json:"from_cache,omitempty"`
	Error     string             `json:"error,omitempty"`
}
