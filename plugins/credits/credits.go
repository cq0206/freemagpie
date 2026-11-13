// freemagpie // Package credits is the Free Credits Market: the catalog
// of free AI credits the FreeMagpie cloud keeps (docs/freemagpie/API.md),
// browsed in the window — searched, filtered, claimed at its official
// page, and, for an offer an API key can serve, added to magpie through
// the import link the cloud hands. The catalog is anonymous and cached,
// so the page works with no account and no network (docs/freemagpie/
// ARCHITECTURE.md §1.2.1: signing in buys sync, not use).
package credits

import (
	"github.com/yetone/magpie/internal/freemagpie"
	"github.com/yetone/magpie/internal/pluginrt"
)

func init() {
	pluginrt.Register(pluginrt.Manifest{
		ID: "credits", API: "^1", Requires: []string{"window", "settings"}, Essential: true, // freemagpie: the fork's own page, on in every preset
		Name: "Free Credits Market", Sub: "Free AI credits of every product, in one list: claim them at their official page, and add the ones with an API to magpie",
		One: "1 offer saved", Many: "{n} offers saved",
		InUse: func() (int, error) {
			rows, err := freemagpie.States()
			if err != nil {
				return 0, err // unknown is in use: a page with nowhere to be seen stays on
			}
			n := 0
			for _, r := range rows {
				if r.State != "dismissed" {
					n++
				}
			}
			return n, nil
		},
		Parts:  []pluginrt.Part{{Slot: "nav.page", Key: "credits"}},
		Client: []string{"i18n.js", "credits.css", "client.js", "page.js"},
	})
}
