# Meridian — Wake-on-LAN

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 ("Wake-on-LAN").

## Intent

An operator wants to power on a machine on their LAN by its MAC address — the
classic Wake-on-LAN "magic packet". Meridian should build the packet correctly,
send it as a UDP broadcast, and explain in plain language what it did and why it
might not wake the target (WoL has to be enabled in BIOS/NIC, and broadcasts
don't cross subnets). This is the §4.6 Wake-on-LAN tool and the first entry in
the otherwise-empty "Local network" tool group beyond ping/scan.

## Why it's safe and in scope

Sending a magic packet is a benign, user-initiated local-network admin action:
it asks one of your own machines to power on. It carries no user identity (just
the target MAC, repeated), reaches only the local broadcast domain, and does
nothing if WoL isn't enabled on the target. No confirmation gate beyond the
button is warranted.

## How Wake-on-LAN works (what the engine encodes)

- **Magic packet.** 6 bytes of `0xFF`, followed by the target's 6-byte MAC
  repeated 16 times — 102 bytes total. Sent as a UDP payload to the broadcast
  address, conventionally port 9 (discard) or 7 (echo).
- **MAC forms.** Accept the common EUI-48 spellings — `aa:bb:cc:dd:ee:ff`,
  `aa-bb-cc-dd-ee-ff`, `aabb.ccdd.eeff`, and bare `aabbccddeeff` — via
  `net.ParseMAC`, requiring a 6-byte address.
- **Broadcast.** Default target `255.255.255.255:9`; the broadcast address and
  port are configurable (a directed subnet broadcast like `192.168.1.255` is
  often needed so the packet reaches the right VLAN). Sending to a broadcast
  address requires the `SO_BROADCAST` socket option, set portably via
  `net.Dialer.Control` with small per-OS helpers.

## Scope

**In scope:**
- Build + validate the magic packet from a MAC (pure, testable).
- Send it as a UDP broadcast to a configurable address/port.
- Engine (`internal/wol`), CLI (`wol`), Wails method, Workbench tool under
  "Local network", all following the established seams and self-explaining UI.

**Non-goals (future / out):**
- SecureOn password (6-byte appended secret) — rarely used; *my call to omit.*
- Routed/WAN WoL (unicast to a router that rebroadcasts) — a later option.
- Discovering or remembering MACs — the input is a MAC the user supplies (the
  IP-space records feature is where saved device MACs will later live).

## UX (Workbench tool, Nielsen-compliant)

- A MAC input, an optional broadcast field (default `255.255.255.255`) and port
  (default 9), and a Wake button. A one-line note explains what a magic packet
  is and the two reasons it may not wake the target (WoL disabled; broadcasts
  don't cross subnets) — the screen explains itself.
- On success, a plain confirmation ("Magic packet sent to <mac> via
  <bcast>:<port>") — honest that "sent" is not "woke": we can't know if the
  machine powered on.
- On a bad MAC or a send error, a clear message.

## Testing posture

- `MagicPacket(mac)` is unit-tested: 102 bytes, the `0xFF` header, the MAC at
  every repetition offset, and rejection of malformed / non-EUI-48 input across
  the accepted spellings.
- `Send` is tested against a **local UDP listener** (127.0.0.1): the datagram it
  receives must equal `MagicPacket(mac)` byte-for-byte. The `SO_BROADCAST`
  option is harmless on a loopback unicast send, so the real send path is
  exercised without actually broadcasting.
