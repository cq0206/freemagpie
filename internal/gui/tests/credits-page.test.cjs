// Run with Node's test runner and Playwright on the module path; see README.md.
// The Free Credits Market page (plugins/credits, freemagpie: docs/freemagpie/
// ARCHITECTURE.md §1): the catalog's cards drawn from the cloud's answer —
// every kind of offer, its reset, Claim at the official page, Add to magpie
// for an API offer; the search field narrows the list; Keep/claimed states
// post to the page's own API; with the cloud off or down, the cached catalog
// serves with its age and the error said beside it, never an empty list
// passed off as fresh. In every language, wide and narrow, Chromium and
// WebKit. The API is faked; nothing reaches a network.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

// two offers, as the cloud sends them (API.md §1): a native credit with an
// expiry, and an API credit with an import link
const offers = (readAt) => ({
  data: [
    { id: "o1", slug: "qoder-daily", product: { slug: "qoder", name: "Qoder", category: "coding-agent" }, title: "Daily Credits", reward_kind: "native_credit", reward_label: "30 a day", reset_period: "daily", claim_method: "manual", claim_url: "https://qoder.example/c", api_compatible: false, verification_status: "official_verified", valid_until: "2027-03-01T00:00:00Z" },
    { id: "o2", slug: "free-api", product: { slug: "example", name: "Example", category: "api-provider" }, title: "Free API tier", reward_kind: "api_credit", reward_label: "1M tokens a day", reset_period: "daily", claim_method: "manual", claim_url: "https://example.example/keys", api_compatible: true, verification_status: "official_verified", import_link: "magpie://import?name=Example&chat=https://api.example.com/v1&models=m1" },
  ],
  read_at: readAt,
});

const words = {
  en: { title: "Free Credits Market", claim: "Get the credits", keep: "Keep this offer", youClaimed: "You claimed it", noMatch: "No offers match", api: "Add to magpie" },
  zh: { title: "免费积分市场", claim: "去领取", keep: "收藏", youClaimed: "已领取", noMatch: "没有符合条件的福利", api: "添加到 magpie" },
  "zh-TW": { title: "免費積分市場", claim: "去領取", keep: "收藏", youClaimed: "已領取", noMatch: "沒有符合條件的福利", api: "新增到 magpie" },
  ja: { title: "無料クレジットマーケット", claim: "受け取る", keep: "保存", youClaimed: "受領済み", noMatch: "該当するオファーはありません", api: "magpie に追加" },
  de: { title: "Kostenlose-Credits-Markt", claim: "Abholen", keep: "Merken", youClaimed: "Abgeholt", noMatch: "Keine Angebote passen", api: "Zu magpie hinzufügen" },
};

// serve fakes the page's API: the catalog (down, or cached with the error
// said beside it), the states, and the window's own pages
function serve(lang, catalog, states, posts) {
  const settings = () => ({ lang, theme: "light" });
  const state = { agents: [], profiles: [], settings: settings() };
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/settings") return json(settings());
    if (url.pathname === "/api/groups") return json({ models: [], groups: [], pools: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true, url: "http://127.0.0.1:3999" } });
    if (url.pathname === "/api/credits/offers") return json(typeof catalog === "function" ? catalog(url) : catalog);
    if (url.pathname === "/api/credits/state") {
      if (r.request().method() === "POST") {
        const body = JSON.parse(r.request().postData());
        posts.push(body);
        states[body.slug] = body.state ? body : undefined; // "" takes it away
        return json(states);
      }
      return json(states);
    }
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await r.fulfill({ body: await fs.readFile(file), contentType });
  };
}

async function open(engine, lang, width, catalog, states, posts) {
  const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
  const page = await (await browser.newContext({ viewport: { width, height: 1400 } })).newPage();
  page.setDefaultTimeout(8000);
  page.setDefaultNavigationTimeout(60000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve(lang, catalog, states, posts));
  await page.goto("http://magpie.test/?view=credits");
  await page.locator("#crdList .crd-offer").first().waitFor();
  return { browser, page, errors };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of Object.keys(words)) {
    for (const width of [1100, 640]) {
      const w = words[lang];
      test(`${engine} ${lang} ${width}px: the catalog's cards, searched, kept and claimed`, async (t) => {
        const states = {};
        const posts = [];
        const { browser, page, errors } = await open(engine, lang, width, offers("2026-11-13T08:00:00Z"), states, posts);
        t.after(() => browser.close());
        const title = await page.evaluate(() => document.querySelector("#pageTitle").textContent);
        assert.equal(title, w.title);

        const cards = page.locator("#crdList .crd-offer");
        assert.equal(await cards.count(), 2, "both offers");
        // the native offer: Claim at its official page, no Add (no API)
        const native = cards.nth(0);
        assert.equal(await native.locator(".crd-actions .go", { hasText: w.claim }).count(), 1, "the native offer says where to claim");
        assert.equal(await native.locator(".crd-actions .go", { hasText: w.api }).count(), 0, "no Add to magpie without an API");
        // the API offer: both
        const api = cards.nth(1);
        assert.equal(await api.locator(".crd-actions .go", { hasText: w.claim }).count(), 1);
        assert.equal(await api.locator(".crd-actions .go", { hasText: w.api }).count(), 1, "an API offer can be added");
        // translated (English keys stay English)
        if (lang !== "en") assert.notEqual(await native.locator(".crd-actions .go").first().textContent(), words.en.claim, "translated");

        // the search narrows the list; a word nothing matches says so
        const q = page.locator("#crdQuery");
        await q.fill("example");
        await page.waitForTimeout(100);
        assert.equal(await page.locator("#crdList .crd-offer").count(), 1, "one card for example");
        await q.fill("nothing-has-this");
        await page.waitForTimeout(100);
        await page.locator("#crdList .empty").waitFor();
        assert.equal(await page.locator("#crdList .empty").textContent(), w.noMatch);
        await q.fill("");
        await page.waitForTimeout(100);
        assert.equal(await page.locator("#crdList .crd-offer").count(), 2, "back to both");

        // Keep this offer posts the state and shows the way back
        await cards.nth(0).locator(".crd-actions .text", { hasText: w.keep }).click();
        await page.waitForTimeout(200);
        assert.deepEqual(posts.map((p) => [p.slug, p.state]), [["qoder-daily", "saved"]], "keep posted");
        // and claiming marks it claimed
        const markWord = { en: "Mark it claimed", zh: "标记为已领取", "zh-TW": "標記為已領取", ja: "受け取ったことにする", de: "Als abgeholt markieren" }[lang];
        await page.locator("#crdList .crd-offer").nth(0).locator(".crd-actions .text", { hasText: markWord }).click();
        await page.waitForTimeout(200);
        assert.ok(posts.some((p) => p.slug === "qoder-daily" && p.state === "claimed"), "claimed posted");
        await page.waitForTimeout(200);
        const card = page.locator("#crdList .crd-offer").first();
        assert.equal(await card.getAttribute("data-state"), "claimed", "the card is marked claimed");
        assert.equal(await card.locator(".crd-actions span", { hasText: w.youClaimed }).count(), 1, "the claimed word shows");

        // nothing runs off the page
        const wide = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
        assert(!wide, "nothing runs off the page");
        assert.deepEqual(errors, [], "no page errors");
      });
    }
  }

  test(`${engine}: the cloud down serves the cache with the error said beside it`, async (t) => {
    const cached = offers("2026-11-13T08:00:00Z");
    const down = { data: cached.data, read_at: cached.read_at, from_cache: true, error: "the cloud is being restarted" };
    const { browser, page, errors } = await open(engine, "en", 1100, down, {}, []);
    t.after(() => browser.close());
    assert.equal(await page.locator("#crdList .crd-offer").count(), 2, "the cached cards still serve");
    const note = await page.locator("#crdNote").textContent();
    assert.match(note, /the cloud is being restarted/, "the error is said");
    assert.deepEqual(errors, [], "no page errors");
  });

  test(`${engine}: no cloud and no cache says so, not an empty catalog passed off as fresh`, async (t) => {
    const gone = { error: "FreeMagpie Cloud is off" };
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    const page = await (await browser.newContext({ viewport: { width: 1100, height: 1400 } })).newPage();
    page.setDefaultTimeout(8000);
    page.setDefaultNavigationTimeout(60000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve("en", gone, {}, []));
    await page.goto("http://magpie.test/?view=credits");
    await page.locator("#crdNote").waitFor();
    assert.equal(await page.locator("#crdList .crd-offer").count(), 0, "no offers");
    const empty = await page.locator("#crdList").textContent();
    assert.notEqual(empty.trim(), "No offers yet", "the empty text is not the cloud-off one"); // the error, not "nothing here"
    const note = await page.locator("#crdNote").textContent();
    assert.match(note, /FreeMagpie Cloud is off/, "the cloud's state is said");
    assert.deepEqual(errors, [], "no page errors");
    await browser.close();
  });
}
