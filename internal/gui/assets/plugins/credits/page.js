// freemagpie // The Free Credits Market page's code: run after app.js,
// whose helpers (el, t, api, openProtoMenu's pick) it shares, before
// start.js shows the first page. It loads the catalog (the cloud's, or
// the cache when the cloud is off or didn't answer — §1.2.1: the page
// opens either way), the user's offer states, and draws the cards:
// search and filters, Get the credits at the official page, Add to
// magpie through the Providers page's import flow, Keep and claimed
// kept locally.
//
// Registered as plugins/credits's page (client.js has its markup); the
// kernel's boundary catches a hook that throws, so nothing here can
// break another page.

// the catalog as last read, and the user's states, by offer slug
let creditsOffers = [];
let creditsStates = {}; // slug → { state, claimed_at }
let creditsError = "";  // the cloud's last error, when the catalog didn't come
let creditsQuery = "";  // the search field's words
let creditsFilters = { kind: "", period: "" };

const CREDITS_KINDS = [
  { v: "", name: "Every kind of offer" },
  { v: "native_credit", name: "Native credits" },
  { v: "api_credit", name: "API credits" },
  { v: "free_model", name: "Offered models" },
  { v: "promo", name: "Promotions" },
];
const CREDITS_PERIODS = [
  { v: "", name: "Any reset" },
  { v: "daily", name: "Resets daily" },
  { v: "weekly", name: "Resets weekly" },
  { v: "monthly", name: "Resets monthly" },
  { v: "none", name: "No reset" },
];

function creditsPageOpen() { return creditsPageLoad(false); }

async function creditsPageLoad(reload) {
  const list = document.getElementById("crdList");
  if (!list) return; // the page isn't there (switched off mid-load)
  list.setAttribute("aria-busy", "true");
  creditsWire(); // the search field and the filters, once
  const [offers, states] = await Promise.all([
    api("credits/offers" + (reload ? "?reload=1" : ""), undefined, undefined, creditsAbort?.signal).catch((err) => ({ error: String(err?.message || err) })),
    api("credits/state").catch(() => ({})),
  ]);
  if (view !== "credits") return; // the reader left while it loaded
  creditsAbort?.abort();
  creditsOffers = offers.data || [];
  creditsStates = states || {};
  creditsError = offers.error || ""; // the note says it; the list stays empty
  list.removeAttribute("aria-busy");
  creditsNote(offers);
  creditsDraw();
}

// creditsAbort stops a load the reader has left behind
let creditsAbort = null;
{
  const c = new AbortController();
  creditsAbort = c;
}

// creditsNote says where the list came from: the cloud's answer, the
// cache (and its age), or the error the cloud gave with the cache
function creditsNote(offers) {
  const note = document.getElementById("crdNote");
  if (!note) return;
  note.hidden = true;
  note.replaceChildren();
  if (offers.error) {
    const err = el("span", "crd-err", t("The cloud couldn't be asked: {err}", { err: offers.error }));
    note.append(err);
    note.hidden = false;
  }
  if (offers.from_cache && offers.read_at) {
    note.append(el("span", "", t("Catalog read {when}", { when: creditsWhen(new Date(offers.read_at)) }) + " · " + t("Shown from the last catalog read; Refresh asks the cloud again")));
    note.hidden = false;
  }
}

// creditsWire puts the search field and the filter menus in, once
function creditsWire() {
  if (creditsWire.done) return;
  creditsWire.done = true;
  const q = document.getElementById("crdQuery");
  if (q) { // its placeholder and label translate with data-tp (i18n.js)
    q.oninput = () => { creditsQuery = q.value.trim().toLowerCase(); creditsDraw(); };
  }
  const filters = document.getElementById("crdFilters");
  if (!filters) return;
  const filterMenu = (opts, pick, onPick) => {
    const b = el("button", "proto pick");
    b.type = "button";
    const paint = () => b.replaceChildren(el("span", "", t(opts.find((o) => o.v === pick)?.name || "")), svg(CHEV, 11, 1.6));
    paint();
    b.onclick = (ev) => {
      ev.stopPropagation();
      openProtoMenu(b, opts.map((o) => ({ ...o, literalName: true, note: "" })), pick, (v) => { pick = v; paint(); onPick(v); }, "", "", "right");
    };
    return b;
  };
  filters.append(
    filterMenu(CREDITS_KINDS, creditsFilters.kind, (v) => { creditsFilters.kind = v; creditsDraw(); }),
    filterMenu(CREDITS_PERIODS, creditsFilters.period, (v) => { creditsFilters.period = v; creditsDraw(); }),
  );
}

// creditsDraw paints the offer cards: the search and filters first,
// then each offer as it stands (claimed ones dimmed)
function creditsDraw() {
  const list = document.getElementById("crdList");
  if (!list) return;
  const q = creditsQuery.split(/\s+/).filter(Boolean);
  const shown = creditsOffers.filter((o) => {
    if (creditsFilters.kind && o.reward_kind !== creditsFilters.kind) return false;
    if (creditsFilters.period && (o.reset_period || "none") !== creditsFilters.period) return false;
    if (q.length) {
      const hay = `${o.title} ${o.product?.name || ""} ${o.product?.slug || ""} ${o.reward_kind}`.toLowerCase();
      if (!q.every((w) => hay.includes(w))) return false;
    }
    return true;
  });
  list.replaceChildren();
  if (!creditsOffers.length) {
    // a cloud that didn't answer is said by the note (creditsNote), not
    // as "no offers" — an empty catalog passed off as fresh it isn't
    if (!creditsError) list.append(el("p", "empty", t("No offers yet")));
    return;
  }
  if (!shown.length) {
    list.append(el("p", "empty", t("No offers match")));
    return;
  }
  for (const o of shown) list.append(creditsCard(o));
}

// creditsCard is one offer: the product, what it gives and when it
// renews, and the actions — Claim at the official page, Add to magpie
// for an API one, Save and Claimed for the user's own state
function creditsCard(o) {
  const card = el("div", "crd-offer");
  const state = creditsStates[o.slug]?.state || "";
  if (state) card.dataset.state = state;
  const who = el("div", "who");
  const name = el("div", "name");
  name.append(el("span", "", o.product?.name || o.product?.slug || ""), el("span", "sub", o.title));
  who.append(name);
  const sub = el("div", "sub", creditsKindName(o));
  who.append(sub);
  const kinds = el("div", "crd-kinds");
  kinds.append(el("span", "crd-kind", o.api_compatible ? "API" : t("In their app")));
  if (o.reset_period && o.reset_period !== "none") kinds.append(el("span", "crd-kind", t(CREDITS_PERIODS.find((p) => p.v === o.reset_period)?.name || o.reset_period)));
  if (o.verification_status === "official_verified") kinds.append(el("span", "crd-kind", t("Verified")));
  else if (o.verification_status === "unverified") kinds.append(el("span", "crd-kind", t("Unverified")));
  if (o.valid_until) kinds.append(el("span", "crd-kind", t("Expires {when}", { when: creditsWhen(new Date(o.valid_until)) })));
  else kinds.append(el("span", "crd-kind", t("Never expires")));
  who.append(kinds);
  card.append(who);

  const actions = el("div", "crd-actions");
  if (o.claim_url) {
    const claim = el("button", "go", t("Get the credits"));
    claim.type = "button";
    claim.onclick = () => window.open(o.claim_url, "_blank", "noopener");
    actions.append(claim);
  }
  // Add to magpie: only an offer the cloud marks API-compatible, and
  // only through the import link the cloud made (provider.ParseImport
  // checked it); the key is the user's to paste in the page that opens
  if (o.api_compatible && o.import_link) {
    const add = el("button", "go", t("Add to magpie"));
    add.type = "button";
    add.onclick = () => api("open", { url: o.import_link });
    actions.append(add);
  }
  if (state === "claimed") {
    actions.append(el("span", "", t("You claimed it")));
  } else if (o.claim_url) {
    const done = el("button", "text", t("Mark it claimed"));
    done.type = "button";
    done.onclick = () => creditsSetState(o.slug, "claimed", new Date().toISOString());
    actions.append(done);
  }
  if (state === "saved") {
    const unsave = el("button", "text", t("Forget this offer"));
    unsave.type = "button";
    unsave.onclick = () => creditsSetState(o.slug, "", null);
    actions.append(unsave);
  } else if (!state) {
    const save = el("button", "text", t("Keep this offer"));
    save.type = "button";
    save.onclick = () => creditsSetState(o.slug, "saved", null);
    actions.append(save);
  }
  card.append(actions);
  return card;
}

// creditsSetState keeps one offer's state of the user's (saved,
// claimed, or "" to take it away) and paints the page again
async function creditsSetState(slug, state, claimedAt) {
  try {
    creditsStates = await api("credits/state", { slug, state, claimedAt }) || {};
  } catch (err) {
    statusErr(err);
    return;
  }
  creditsDraw();
}

// creditsKindName says what an offer gives, with its label when it has
// one ("Daily credits · 30 credits a day")
function creditsKindName(o) {
  const kind = t(creditsKindMap[o.reward_kind] || o.reward_kind);
  return o.reward_label ? `${kind} · ${o.reward_label}` : kind;
}

const creditsKindMap = {
  native_credit: "Native credits",
  api_credit: "API credits",
  free_model: "Offered models",
  promo: "Promotions",
};

// creditsWhen says a time shortly: today, or the date
function creditsWhen(d) {
  if (!d || isNaN(d)) return "";
  const now = new Date();
  if (d.toDateString() === now.toDateString()) return t("Today");
  return d.toLocaleDateString();
}
