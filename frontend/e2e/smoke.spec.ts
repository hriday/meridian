import { test, expect } from "@playwright/test";
import { stubWails } from "./stub";

// Console noise that isn't a real UI failure.
const BENIGN = [/favicon/i, /ERR_CONNECTION/i, /net::/i, /Failed to load resource/i];
const benign = (s: string) => BENIGN.some((re) => re.test(s));

test.beforeEach(async ({ page }) => {
  await page.addInitScript(stubWails);
});

test("every tool mounts and renders without a console error", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error" && !benign(m.text())) errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));

  await page.goto("/");
  await page.waitForSelector("nav.side .item");

  const items = page.locator("nav.side .item");
  const count = await items.count();
  expect(count, "the sidebar should list the whole toolset").toBeGreaterThan(60);

  const failures: string[] = [];
  for (let i = 0; i < count; i++) {
    const item = items.nth(i);
    const name = ((await item.textContent()) || `#${i}`).trim();
    const mark = errors.length;

    await item.click();
    await page.locator("main.main").waitFor();
    await page.waitForTimeout(30); // let the component settle

    const html = (await page.locator("main.main").innerHTML()).trim();
    if (html.length < 20) failures.push(`${name}: pane rendered empty`);
    const fresh = errors.slice(mark);
    if (fresh.length) failures.push(`${name}: ${fresh.join(" | ")}`);
  }

  expect(failures, `\n${failures.join("\n")}\n`).toEqual([]);
});

// Request/response tools whose result templates (which only render once backend
// data arrives) should be exercised, not just their empty mount state. The stub
// feeds each an empty-but-iterable mock, so the result path renders without data
// — catching "x is not iterable" / undefined-field crashes in result markup.
const RUN_TOOLS = [
  "lookup", "subnet", "dnsaudit", "axfr", "openresolver",
  "emailauth", "cipherscan", "certexpiry", "sechdr", "http3", "whois",
];

test("key tools run their action and render results without error", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error" && !benign(m.text())) errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));

  await page.goto("/");
  await page.waitForSelector("nav.side .item");

  const failures: string[] = [];
  for (const id of RUN_TOOLS) {
    const mark = errors.length;
    await page.locator(`[data-tool="${id}"]`).click();
    await page.locator("main.main").waitFor();

    const input = page.locator("main.main input[type='text'], main.main input:not([type])").first();
    if (await input.count()) {
      await input.fill("example.com").catch(() => {});
    }
    const btn = page.locator("main.main button:not([disabled])").first();
    if (await btn.count()) {
      await btn.click().catch(() => {});
    }
    await page.waitForTimeout(80);

    const html = (await page.locator("main.main").innerHTML()).trim();
    if (html.length < 20) failures.push(`${id}: pane blanked after run`);
    const fresh = errors.slice(mark);
    if (fresh.length) failures.push(`${id}: ${fresh.join(" | ")}`);
  }

  expect(failures, `\n${failures.join("\n")}\n`).toEqual([]);
});

test("the sidebar filter narrows the tool list", async ({ page }) => {
  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  const all = await page.locator("nav.side .item").count();

  const filter = page.locator("input.search");
  await filter.fill("dns");
  await page.waitForTimeout(100);
  const narrowed = await page.locator("nav.side .item").count();

  expect(narrowed).toBeGreaterThan(0);
  expect(narrowed).toBeLessThan(all);
});

test("service status: a user can pin and remove a custom service chip", async ({ page }) => {
  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="svcstatus"]').click();
  await page.locator("main.main").waitFor();

  // Open the add form, enter a name + URL, save.
  await page.getByRole("button", { name: "+ Add service" }).click();
  await page.getByPlaceholder("name (optional)").fill("Acme");
  await page.getByPlaceholder(/status URL/).fill("status.acme.com");
  await page.getByRole("button", { name: "Save", exact: true }).click();

  // The chip is pinned.
  const chip = page.locator(".chip.custom", { hasText: "Acme" });
  await expect(chip).toHaveCount(1);

  // It persists across a reload (localStorage).
  await page.reload();
  await page.locator('[data-tool="svcstatus"]').click();
  await expect(page.locator(".chip.custom", { hasText: "Acme" })).toHaveCount(1);

  // Removing it takes it away.
  await page.locator(".chip.custom", { hasText: "Acme" }).locator(".cx").click();
  await expect(page.locator(".chip.custom", { hasText: "Acme" })).toHaveCount(0);
});

test("sidebar search matches tool keywords across the toolset, not just names", async ({ page }) => {
  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  const filter = page.locator("input.search");

  // A broad spread of concept/synonym searches, none of which appear in the
  // tool's visible name — each must still surface the right tool.
  const cases: [string, string][] = [
    ["rpki", "bgp"],
    ["rdap", "lookup"],
    ["dig", "dns"],
    ["nslookup", "dns"],
    ["ssl", "tls"],
    ["spf", "emailauth"],
    ["dkim", "emailauth"],
    ["dnssec", "dnsaudit"],
    ["iperf", "speedtest"],
    ["nmap", "ports"],
    ["base64", "transcode"],
    ["haveibeenpwned", "breaches"],
    ["terraform", "tfexposure"],
    ["oui", "macvendor"],
    ["statuspage", "svcstatus"],
  ];
  for (const [term, id] of cases) {
    await filter.fill(term);
    await page.waitForTimeout(50);
    await expect(page.locator(`nav.side [data-tool="${id}"]`), `"${term}" should surface ${id}`).toBeVisible();
  }

  // Multi-word queries narrow by every term (token AND).
  await filter.fill("post quantum");
  await page.waitForTimeout(50);
  await expect(page.locator('nav.side [data-tool="sshaudit"]')).toBeVisible();

  // A term in no tool still yields nothing (no false matches).
  await filter.fill("zzxqnotathing");
  await page.waitForTimeout(50);
  await expect(page.locator("nav.side .item")).toHaveCount(0);
});

test("settings: the gear opens Settings and the theme choice persists", async ({ page }) => {
  await page.goto("/");
  await page.waitForSelector("nav.side .item");

  await page.locator("button.gear").click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog).toBeVisible();

  // Pick Dark — it applies immediately via data-theme on <html>.
  await dialog.getByRole("radio", { name: /Dark/ }).click();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.dataset.theme))
    .toBe("dark");

  // Escape closes it, and the choice survives a reload (localStorage).
  await page.keyboard.press("Escape");
  await expect(dialog).not.toBeVisible();
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.dataset.theme))
    .toBe("dark");
});

test("wwn decoder renders a result card for an input", async ({ page }) => {
  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="wwn"]').click();
  await page.locator("main.main").waitFor();
  const pane = page.locator("main.main");
  await pane.locator('input[aria-label="WWN / WWPN"]').fill("20:00:00:25:b5:00:00:01");
  await pane.getByRole("button", { name: "Decode" }).click();
  await expect(pane.locator(".card")).toBeVisible();
});

test("fc zoning analyzer accepts a pasted config without a render crash", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="fczone"]').click();
  const pane = page.locator("main.main");
  await pane.waitFor();
  await pane.locator("textarea").fill("Defined configuration:\n cfg: C1 Z1\n zone: Z1 a; b; c\n");
  await pane.getByRole("button", { name: "Analyze" }).click();
  await page.waitForTimeout(120);
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("fc zoning analyzer draws the zone map from real data", async ({ page }) => {
  // Give this page a bridge that returns a concrete report for AnalyzeFCZoning,
  // so the SVG actually renders (the default stub returns empty proxies).
  await page.addInitScript(() => {
    const REAL = {
      vendor: "Brocade FOS",
      counts: { zonesets: 1, zones: 3, aliases: 1, unparsed: 0 },
      zonesets: [{ name: "C1", active: true, zones: ["Z1", "Z2", "Z3"] }],
      zones: [
        { name: "Z1", members: [{ kind: "wwn", value: "21:00:00:1b:32:04:86:c3" }, { kind: "wwn", value: "10:00:00:00:c9:3e:4c:eb" }] },
        { name: "Z2", members: [{ kind: "wwn", value: "21:00:00:1b:32:04:86:c3" }, { kind: "wwn", value: "50:06:01:60:88:02:90:cb" }, { kind: "wwn", value: "50:06:01:61:88:02:90:cb" }] },
        { name: "Z3", members: [{ kind: "name", value: "aliasX" }] },
      ],
      aliases: [{ name: "aliasX", members: [] }],
      findings: [],
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) {
        if (p === "then") return undefined;
        if (p === Symbol.iterator) return function* () {};
        if (p === "length") return 0;
        if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => "";
        return bridge(path.concat(String(p)));
      },
      apply: () => (path.includes("AnalyzeFCZoning") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "fczone"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="fczone"]').click();
  const pane = page.locator("main.main");
  await pane.locator("textarea").fill("anything");
  await pane.getByRole("button", { name: "Analyze" }).click();

  const svg = pane.locator("svg.map");
  await expect(svg).toBeVisible();
  await expect(svg.locator(".node.z")).toHaveCount(3);   // 3 zones
  await expect(svg.locator(".node.m")).toHaveCount(5);   // 5 unique members
  await expect(svg.locator(".edge")).toHaveCount(6);     // 2+3+1 memberships
  await expect(svg.locator(".zlabel.fat")).toHaveCount(1);    // Z2 has 3 members
  await expect(svg.locator(".mlabel.shared").first()).toBeVisible(); // 21:..c3 is in 2 zones
});

test("printer status renders state, supplies and ports from a real report", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = {
      host: "10.0.0.50", reachable: true, name: "lobby", makeModel: "ACME LaserJet 9000", location: "Lobby",
      state: "stopped", stateReasons: ["media-empty", "toner-low"], queuedJobs: 1, ippVersions: ["2.0"],
      supplies: [
        { name: "Black Toner", level: 8, color: "#000000", type: "toner", max: 100 },
        { name: "Cyan", level: -1, color: "#00ffff", max: 0 },
      ],
      ports: [{ port: 631, name: "IPP", open: true }, { port: 9100, name: "raw / JetDirect", open: true }, { port: 515, name: "LPD", open: false }],
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("PrinterStatus") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "printer"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="printer"]').click();
  const pane = page.locator("main.main");
  await pane.locator('input[aria-label="Printer host"]').fill("10.0.0.50");
  await pane.getByRole("button", { name: "Check" }).click();

  await expect(pane.locator(".pname")).toHaveText("ACME LaserJet 9000");
  await expect(pane.locator(".state.bad")).toBeVisible();        // stopped
  await expect(pane.locator(".reason")).toHaveCount(2);          // media-empty, toner-low
  await expect(pane.locator(".supply")).toHaveCount(2);
  // The toner colour renders as a swatch (black toner -> black swatch, not faked).
  await expect(pane.locator(".supply").first().locator(".swatch")).toHaveCSS("background-color", "rgb(0, 0, 0)");
  await expect(pane.locator(".fill.low")).toHaveCount(1);        // 8% toner
  await expect(pane.locator(".supply .spct.unk")).toHaveText("level unknown");
  await expect(pane.locator(".port.open")).toHaveCount(2);       // 631 + 9100
});
