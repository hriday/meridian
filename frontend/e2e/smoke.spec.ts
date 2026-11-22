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
  "emailauth", "cipherscan", "certexpiry", "sechdr", "http3", "whois", "mesh", "ble", "peripherals", "linkpath",
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
    // Skip the ToolHelp "How to read this" disclosure (`.chip`), which is the
    // first button in most tools — clicking it would toggle help and never run
    // the action, so the result block (where render crashes hide) never mounts.
    const btn = page.locator("main.main button:not([disabled]):not(.chip)").first();
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

test("the IP chip shows local + public IP and expands to interfaces, gateway and NAT", async ({ page }) => {
  await page.addInitScript(() => {
    const LOCAL = {
      primaryIP: "192.168.1.82", primaryIf: "en0", gateway: "192.168.1.1",
      interfaces: [{ name: "en0", ipv4: ["192.168.1.82"] }, { name: "en5", ipv4: ["10.0.0.3"] }],
    };
    const PUBLIC = { publicIP: "203.0.113.45", natType: "Full cone (endpoint-independent)", nat: true, localIP: "192.168.1.82", publicPort: 54321 };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) {
        if (p === "then") return undefined;
        if (p === Symbol.iterator) return function* () {};
        if (p === "length") return 0;
        if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => "";
        return bridge(path.concat(String(p)));
      },
      apply: () => path.includes("LocalNetwork") ? Promise.resolve(LOCAL)
        : path.includes("STUNDiscover") ? Promise.resolve(PUBLIC)
        : Promise.resolve(perm()),
    });
    (window as any).go = bridge([]);
  });

  await page.goto("/");
  const chip = page.locator(".ipchip .chip");
  await expect(chip).toContainText("192.168.1.82");  // local, instant
  await expect(chip).toContainText("203.0.113.45");  // public, after STUN resolves

  await chip.click();
  const pop = page.locator(".ipchip .pop");
  await expect(pop).toBeVisible();
  await expect(pop).toContainText("en0");
  await expect(pop).toContainText("10.0.0.3");       // the second interface
  await expect(pop).toContainText("192.168.1.1");    // gateway
  await expect(pop).toContainText("Full cone");      // NAT type
});

test("security headers renders the scorecard, rows and remediation from a real report", async ({ page }) => {
  // The generic RUN_TOOLS pass drives sechdr with an empty proxy; this one feeds
  // a concrete Report so the full result markup renders — including the
  // "How to improve" block, which referenced an undefined `rep` and crashed the
  // render for anyone who ran a real scan (shipped unseen; Go-only CI + the
  // proxy stub both missed it).
  await page.addInitScript(() => {
    const REAL = {
      url: "https://example.com", finalUrl: "https://example.com/", https: true, status: 200,
      score: 72, grade: "B",
      headers: [
        { name: "Strict-Transport-Security", present: true, value: "max-age=63072000", grade: "ok", note: "HSTS set with a long max-age.", owasp: "A05 Security Misconfiguration" },
        { name: "Content-Security-Policy", present: false, value: "", grade: "bad", note: "No CSP — the strongest XSS defence is absent.", owasp: "A03 Injection" },
      ],
      remediation: ["Content-Security-Policy: default-src 'self'", "Permissions-Policy: geolocation=(), camera=()"],
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("SecurityHeaders") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "sechdr"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="sechdr"]').click();
  const pane = page.locator("main.main");
  await pane.locator("input").first().fill("example.com");
  await pane.getByRole("button", { name: "Scan headers" }).click();

  await expect(pane.locator(".scorecard .grade")).toHaveText("B");
  await expect(pane.locator(".hrow")).toHaveCount(2);              // the two header rows
  await expect(pane.locator(".improve")).toBeVisible();           // the block that was crashing
  await expect(pane.locator(".improve .fix")).toHaveCount(2);     // both remediation lines
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("meshtastic renders the node header and mesh rows from a real snapshot", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = {
      transport: "tcp", target: "192.168.1.50:4403", myNodeNum: 100,
      firmware: "2.5.3.abcdef", hwModel: "HELTEC_V3",
      me: { num: 100, id: "!00000064", longName: "Base", shortName: "BASE", role: "Client", snr: 0, hopsAway: 0, lastHeard: 0, viaMqtt: false, battery: 101, voltage: 4.1, hasPosition: true, lat: 37.49, lon: -122.29, altitudeM: 10 },
      nodes: [
        { num: 200, id: "!000000c8", longName: "Hilltop", shortName: "HILL", role: "Client", snr: 6.3, hopsAway: 1, lastHeard: Math.floor(Date.now() / 1000) - 120, viaMqtt: false, battery: 72, voltage: 3.9, hasPosition: true, lat: 37.5, lon: -122.3, altitudeM: 84 },
        { num: 300, id: "!0000012c", longName: "Ridge", shortName: "", role: "", snr: -11, hopsAway: 3, lastHeard: 0, viaMqtt: false, battery: null, voltage: 0, hasPosition: false, lat: 0, lon: 0, altitudeM: 0 },
      ],
      channelUtil: 12.5, airUtilTx: 3.2, nodeCount: 3, takenAt: "2026-11-20T00:00:00Z",
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("MeshtasticSnapshot") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "mesh"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  // Serve a 1x1 transparent PNG for every map tile so the test never hits the
  // real OpenStreetMap servers (and no tile-load error trips the console check).
  const PIXEL = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==", "base64");
  await page.route(/tile\.openstreetmap\.org/, (route) => route.fulfill({ status: 200, contentType: "image/png", body: PIXEL }));

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="mesh"]').click();
  const pane = page.locator("main.main");
  await pane.locator("input").first().fill("192.168.1.50");
  await pane.getByRole("button", { name: "Read node" }).click();

  await expect(pane.locator(".nodehead .nname")).toHaveText("Base");
  await expect(pane.locator(".noderow")).toHaveCount(2);         // two neighbours
  await expect(pane.locator(".noderow").first()).toContainText("Hilltop");
  await expect(pane.locator(".noderow").first().locator("a")).toHaveText("37.5000, -122.3000"); // position link

  // The map is opt-in: Leaflet does not run until "Show map" is clicked (it froze
  // the macOS webview when it auto-initialised). Before clicking, there is no map.
  await expect(pane.locator(".mapbox")).toHaveCount(0);
  await pane.getByRole("button", { name: "Show map" }).click();

  // After opting in, Leaflet renders a marker per positioned node: "my node"
  // (Base) + Hilltop = 2, with Base as the distinct "me" marker.
  await expect(pane.locator(".mapbox.leaflet-container")).toBeVisible(); // Leaflet adds the class to the div itself
  await expect(pane.locator(".mk")).toHaveCount(2);
  await expect(pane.locator(".mk.me")).toHaveCount(1);
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("meshtastic lists attached USB serial ports and reads one on click", async ({ page }) => {
  await page.addInitScript(() => {
    const PORTS = [{ path: "/dev/cu.usbmodem1101", label: "Espressif Systems LilyGO T-Deck" }];
    const SNAP = {
      transport: "serial", target: "/dev/cu.usbmodem1101", myNodeNum: 100,
      firmware: "2.5.3", hwModel: "TDECK",
      me: { num: 100, id: "!00000064", longName: "Base", shortName: "BASE", role: "Client", snr: 0, hopsAway: 0, lastHeard: 0, viaMqtt: false, battery: 101, voltage: 4.1, hasPosition: false, lat: 0, lon: 0, altitudeM: 0 },
      nodes: [], channelUtil: 5, airUtilTx: 1, nodeCount: 1, takenAt: "2026-11-20T00:00:00Z",
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => {
        if (path.includes("MeshtasticSerialPorts")) return Promise.resolve(PORTS);
        if (path.includes("MeshtasticSnapshot")) return Promise.resolve(SNAP);
        return Promise.resolve(perm());
      },
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "mesh"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="mesh"]').click();
  const pane = page.locator("main.main");

  // The attached board is offered as a pickable chip — label and device path.
  const chip = pane.locator(".picker .chip");
  await expect(chip).toHaveCount(1);
  await expect(chip).toContainText("Espressif Systems LilyGO T-Deck");
  await expect(chip).toContainText("/dev/cu.usbmodem1101");

  // Clicking it fills the target and reads that node — one click, no path typed.
  await chip.click();
  await expect(pane.locator(".nodehead .nname")).toHaveText("Base");
  await expect(pane.locator("input").first()).toHaveValue("/dev/cu.usbmodem1101");
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("ble survey renders the device list from a real scan", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = {
      durationSec: 6, count: 3, named: 2,
      devices: [
        { address: "11111111-1111-1111-1111-111111111111", name: "AirPods Pro", rssi: -48, company: "Apple", seen: 9 },
        { address: "22222222-2222-2222-2222-222222222222", name: "Fit Watch", rssi: -67, company: "Garmin", seen: 4 },
        { address: "33333333-3333-3333-3333-333333333333", name: "", rssi: -88, company: "0x0157", seen: 1 },
      ],
      takenAt: "2026-11-20T00:00:00Z",
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("BLESurvey") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "ble"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="ble"]').click();
  const pane = page.locator("main.main");
  await pane.getByRole("button", { name: "Scan" }).click();

  await expect(pane.locator(".drow")).toHaveCount(3);
  await expect(pane.locator(".drow").first()).toContainText("AirPods Pro"); // strongest first
  await expect(pane.locator(".drow").first()).toContainText("Apple");
  await expect(pane.locator(".drow").last()).toContainText("(unnamed)");    // the nameless device
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("connected peripherals renders the device list and security flags from a real survey", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = {
      count: 4, flagged: 3, takenAt: "2026-11-21T00:00:00Z",
      devices: [
        { name: "Ducky", vendor: "", vendorId: "0x8888", productId: "0x2222", kind: "Keyboard", receiver: false, input: true,
          flags: [
            { level: "watch", text: "Combines a keyboard with storage — that's the shape of a BadUSB / \"rubber ducky\" device." },
            { level: "watch", text: "Unrecognized vendor — confirm you know what it is." },
          ] },
        { name: "USB Keyboard", vendor: "", vendorId: "0x9999", productId: "0x1111", kind: "Keyboard", receiver: false, input: true,
          flags: [{ level: "watch", text: "A keyboard from an unrecognized vendor." }] },
        { name: "USB Receiver", vendor: "Logitech", vendorId: "0x046d", productId: "0xc52b", kind: "Wireless receiver", receiver: true, input: false,
          flags: [{ level: "info", text: "Wireless receiver — it can have input devices paired to it." }] },
        { name: "Ultra Fit", vendor: "SanDisk", vendorId: "0x0781", productId: "0x5583", kind: "Storage", receiver: false, input: false },
      ],
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("PeripheralsSurvey") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "peripherals"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="peripherals"]').click();
  const pane = page.locator("main.main");
  await pane.getByRole("button", { name: "Scan" }).click();

  await expect(pane.locator(".prow")).toHaveCount(4);
  await expect(pane.locator(".prow").first()).toContainText("Ducky");          // watch-flagged ranks first
  await expect(pane.locator(".prow").first()).toContainText("BadUSB");
  await expect(pane.locator(".prow").first().locator(".flag.watch")).toHaveCount(2);
  await expect(pane.locator(".co.unk").first()).toBeVisible();                 // unknown-vendor badge
  await expect(pane.locator(".flag.info")).toHaveCount(1);                     // the receiver's info flag
  await expect(pane.locator(".prow").last()).toContainText("SanDisk");         // clean device ranks last
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("link inspector renders the verdict and findings from a real report", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = {
      iface: "en5", gateway: "192.168.1.1", wireless: false, linkKnown: true,
      speedMbps: 100, duplex: "half", media: "100baseTX", lldpStatus: "needs-privilege",
      neighbors: [],
      verdict: "A hub is almost certainly in the path (the link is half-duplex).",
      findings: [
        { level: "found", text: "Half-duplex link (100 Mb/s, half duplex). A hub shares the medium and forces half-duplex." },
        { level: "limit", text: "Couldn't listen for switch advertisements — raw capture needs elevated privilege." },
        { level: "limit", text: "An unmanaged switch is invisible to every check here." },
      ],
      takenAt: "2026-11-21T00:00:00Z",
    };
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("LinkInspect") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.activeTool", "linkpath"); localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  await page.locator('[data-tool="linkpath"]').click();
  const pane = page.locator("main.main");
  await pane.getByRole("button", { name: "Inspect link" }).click();

  await expect(pane.locator(".verdict.warn")).toContainText("hub is almost certainly"); // hub verdict → amber
  await expect(pane.locator(".findrow")).toHaveCount(3);
  await expect(pane.locator(".findrow").first()).toContainText("Half-duplex");
  await expect(pane.locator(".fmark.found").first()).toBeVisible();
  await expect(pane.locator(".facts")).toContainText("192.168.1.1");               // gateway fact
  expect(errors, errors.join(" | ")).toEqual([]);
});

test("tools overview opens from the footer and filters by name or purpose", async ({ page }) => {
  await page.addInitScript(() => {
    const REAL = [
      { name: "Local network", tools: [
        { name: "Link inspector", desc: "Work out whether there's a hub or switch between this machine and the router." },
        { name: "Connected peripherals", desc: "Inventory what's plugged into USB, with honest security call-outs." },
      ] },
      { name: "Security", tools: [
        { name: "TLS cipher scan", desc: "See which TLS versions and cipher suites a server accepts, graded." },
      ] },
    ];
    const perm = (): any => new Proxy(function () {}, { get(_t, p) { if (p === Symbol.iterator) return function* () {}; if (p === "then") return undefined; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return perm(); }, apply: () => perm() });
    const bridge = (path: string[]): any => new Proxy(function () {}, {
      get(_t, p) { if (p === "then") return undefined; if (p === Symbol.iterator) return function* () {}; if (p === "length") return 0; if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => ""; return bridge(path.concat(String(p))); },
      apply: () => (path.includes("GUIReference") ? Promise.resolve(REAL) : Promise.resolve(perm())),
    });
    (window as any).go = bridge([]);
    try { localStorage.setItem("meridian.closedGroups", "{}"); } catch {}
  });

  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

  await page.goto("/");
  await page.waitForSelector("nav.side .item");
  // The overview lives in the sidebar footer (a help page), not the tool list.
  await page.locator("nav.side .sidefoot").getByRole("button", { name: "Tools overview", exact: true }).click();
  const pane = page.locator("main.main");

  await expect(pane.getByRole("button", { name: /Export tools overview/ })).toBeVisible();
  await expect(pane.locator(".trow")).toHaveCount(3);                               // all tools shown
  await expect(pane.locator(".group")).toHaveCount(2);                              // two groups

  // Filtering on a word from a description, not a name, still finds the tool.
  await pane.locator("input.field").fill("router");
  await expect(pane.locator(".trow")).toHaveCount(1);
  await expect(pane.locator(".trow").first()).toContainText("Link inspector");
  expect(errors, errors.join(" | ")).toEqual([]);
});
