# Security Policy

## Reporting a vulnerability

Please report security issues **privately** using GitHub's
**Security → Advisories → "Report a vulnerability"** on this repository, rather
than opening a public issue. I'll acknowledge the report and work on a fix.

## Scope and posture

Meridian runs **entirely on your machine and hosts nothing for anyone else.**
Its outbound requests go only to the public services you explicitly invoke —
RDAP, RIPEstat, Team Cymru (IP-to-ASN), and the NTP/DNS servers you query —
with a generic `meridian/<version>` User-Agent and **no identifying
information**. The IP-space inventory and saved audit reports are encrypted at
rest with a passphrase you choose.

The **AD password audit**, **netcat**, and the **ping sweep / port scanner** are
**defensive, self-use diagnostics** for networks and systems you administer.
They contain no stealth, evasion, or attack features. Use them only where you
are authorized to do so.
