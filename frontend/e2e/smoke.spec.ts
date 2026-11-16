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
