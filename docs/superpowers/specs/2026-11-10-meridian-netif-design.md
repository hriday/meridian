# Meridian — Network interfaces viewer

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 (local
tooling), §4.6 "Local / environment" group (NTP, DNS already present).

## Intent

A quick, honest answer to "what network interfaces does this machine have, and
how are they configured?" — each NIC's name, MAC address, MTU, up/down and other
flags, and its IPv4/IPv6 addresses. The everyday first thing you check when a
box's networking is misbehaving, without leaving the app for `ifconfig` / `ip a`
/ `ipconfig`. It pairs with the MAC-vendor tool (copy a NIC's MAC to identify its
maker).

## Why it's safe and in scope

Entirely local and read-only: it reads this machine's own interface list via the
Go standard library (`net.Interfaces()`), makes no network connection, and sends
nothing anywhere. A "Local / environment" tool beside NTP and DNS.

## What it shows (per interface)

- **Name** (e.g. `en0`, `eth0`, `lo0`) and **index**.
- **MAC** (hardware address) when the interface has one — formatted
  `aa:bb:cc:dd:ee:ff`; empty for interfaces without one (loopback, some tunnels).
- **MTU**.
- **Flags** in plain words: up / down, loopback, broadcast, point-to-point,
  multicast, running.
- **Addresses**, split into IPv4 and IPv6 (in CIDR form, e.g. `192.168.1.20/24`,
  `fe80::1/64`).

## Behaviour

- Interfaces are listed in a stable order (by index). An interface whose address
  enumeration fails is still listed (name/MAC/flags), with its addresses empty —
  one bad interface never hides the rest.
- No filtering by default (down and loopback interfaces are shown, clearly
  marked) — the tool is a faithful mirror of the OS view.

## Scope

**In scope:** read and present the local interface list (stdlib, cross-platform);
engine (`internal/netif`), CLI (`interfaces`), Wails method, and a Workbench tool
under "Local / environment".

**Non-goals (future / out):** changing interface configuration; per-interface
traffic counters / bandwidth (a monitoring concern); auto-identifying each MAC's
vendor (that's one network call per NIC — the user copies a MAC into the MAC-
vendor tool instead); routing table (a separate "route viewer" slice). *My calls.*

## UX (Workbench tool, Nielsen-compliant)

- No input — it just lists the interfaces on load, with a Refresh button and a
  one-line note on what it shows.
- One card per interface: name + up/down state prominent, MAC and MTU, flag
  chips, and the IPv4/IPv6 addresses. Down/loopback interfaces visually muted.
- The screen explains itself; nothing leaves the machine.

## Testing posture

- The assembly from `net.Interface` + addresses into the view model is pure and
  tested behind two seams (the interface list and the per-interface address
  lookup): a fake interface with a MAC, flags, and mixed v4/v6 addresses asserts
  the formatting and the v4/v6 split; a fake whose address lookup errors asserts
  the interface is still listed with empty addresses and no overall error.
