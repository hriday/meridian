//go:build linux

package trace

// On Linux the unprivileged ICMP "ping" socket only surfaces errors for its own
// echoes, not the Time-Exceeded that quote our TCP SYNs — so a TCP traceroute
// needs a raw ICMP socket (root/sudo) to read the per-hop replies. (A future
// enhancement could use IP_RECVERR's socket error queue to do this unprivileged.)
const tcpICMPNetwork = "ip4:icmp"
