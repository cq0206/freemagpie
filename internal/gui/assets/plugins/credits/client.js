// freemagpie // The Free Credits Market page: the catalog of free AI
// credits (docs/freemagpie/API.md), its words in i18n.js beside this
// file. The page registers before app.js and draws itself in mount:
// a search field, the filters, and the offer cards. Claim opens the
// official page; Add to magpie hands the offer's import link to the
// Providers page's own confirm flow (provider.ParseImport's); Save and
// Claimed keep the offer's state of the user's, locally until they
// sign in (§1.2.1). With the cloud unreachable the cached catalog is
// shown, its age and the error said beside it.
magpie.plugin("credits", (ctx) => {
  ctx.slots.add("page", {
    key: "credits",
    title: "Free Credits Market",
    html: `
  <section class="crd-head">
    <div class="crd-search">
      <input class="sess-filter" type="search" id="crdQuery" spellcheck="false" autocomplete="off" placeholder="Search offers, products" aria-label="Search offers, products" data-tp>
    </div>
    <div class="crd-filters" id="crdFilters"></div>
  </section>
  <section class="crd-note" id="crdNote" hidden></section>
  <section class="crd-list" id="crdList" aria-busy="true"></section>`,
    open() { return creditsPageOpen(); },
    reload() { return creditsPageLoad(true); },
  });
});
