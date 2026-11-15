//go:build darwin

package trace

// macOS delivers the ICMP Time-Exceeded that quote our TCP SYNs to the
// unprivileged ICMP datagram socket (verified), so a TCP traceroute needs no
// root here — same socket the ICMP traceroute already uses without privilege.
const tcpICMPNetwork = "udp4"
