// Mirrors internal/netinput.Kind (iota order). Wails exposes Kind as a
// number; this keeps the frontend readable and in one place.
export enum Kind {
  Unknown = 0,
  IPv4 = 1,
  IPv6 = 2,
  PrefixV4 = 3,
  PrefixV6 = 4,
  ASN = 5,
  Domain = 6,
  Hostname = 7,
}

export function isPrefix(k: number): boolean {
  return k === Kind.PrefixV4 || k === Kind.PrefixV6;
}

export function kindLabel(k: number): string {
  switch (k) {
    case Kind.IPv4: return "IPv4 address";
    case Kind.IPv6: return "IPv6 address";
    case Kind.PrefixV4: return "IPv4 prefix";
    case Kind.PrefixV6: return "IPv6 prefix";
    case Kind.ASN: return "AS number";
    case Kind.Domain: return "domain";
    case Kind.Hostname: return "hostname";
    default: return "value";
  }
}
