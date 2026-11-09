import { writable } from "svelte/store";

// A request to switch the active tool, optionally prefilling it with a value
// (used by IP-space records to pivot a record into Subnet calc / Lookup).
export const requestedTool = writable<{ tool: string; value?: string } | null>(null);
