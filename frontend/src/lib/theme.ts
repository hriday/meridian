// Theme preference: Auto (follow OS), Light, or Dark. The choice is stored
// per-machine in localStorage; "auto" resolves against the OS appearance and
// tracks it live. applyTheme sets data-theme on <html>, which the CSS tokens
// in App.svelte switch on.
import { writable } from "svelte/store";

export type ThemePref = "auto" | "light" | "dark" | "terminal";
type Effective = "light" | "dark" | "terminal";
const KEY = "meridian.theme";
const ORDER: ThemePref[] = ["auto", "light", "dark", "terminal"];

export const themeIcon: Record<ThemePref, string> = {
  auto: "◐",
  light: "☀",
  dark: "☾",
  terminal: "▮",
};
export const themeLabel: Record<ThemePref, string> = {
  auto: "Auto",
  light: "Light",
  dark: "Dark",
  terminal: "Terminal",
};

const mql =
  typeof window !== "undefined" && window.matchMedia
    ? window.matchMedia("(prefers-color-scheme: light)")
    : null;

function load(): ThemePref {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "light" || v === "dark" || v === "auto") return v;
  } catch {
    /* private mode / blocked storage: fall through to default */
  }
  return "auto";
}

function effective(pref: ThemePref): Effective {
  if (pref === "auto") return mql && mql.matches ? "light" : "dark";
  return pref;
}

function apply(pref: ThemePref) {
  try {
    document.documentElement.dataset.theme = effective(pref);
  } catch {
    /* no document (SSR/tests): nothing to apply */
  }
}

let current: ThemePref = load();
export const themePref = writable<ThemePref>(current);

// Persist and apply on every change; the first (immediate) call applies the
// stored preference at import time, before Svelte mounts.
themePref.subscribe((p) => {
  current = p;
  try {
    localStorage.setItem(KEY, p);
  } catch {
    /* ignore */
  }
  apply(p);
});

// When following the OS, re-resolve as the OS appearance flips.
if (mql) {
  const onChange = () => {
    if (current === "auto") apply(current);
  };
  if (mql.addEventListener) mql.addEventListener("change", onChange);
  else if ((mql as unknown as { addListener?: (f: () => void) => void }).addListener)
    (mql as unknown as { addListener: (f: () => void) => void }).addListener(onChange);
}

export function cycleTheme() {
  themePref.update((p) => ORDER[(ORDER.indexOf(p) + 1) % ORDER.length]);
}
