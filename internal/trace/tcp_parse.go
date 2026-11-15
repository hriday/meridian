package trace

import "encoding/binary"

// quotedTCPPorts pulls the source and destination TCP ports out of the packet
// quoted inside an ICMP error (Time Exceeded / Destination Unreachable). The
// quoted payload is the original IPv4 header followed by at least the first 8
// bytes of the original datagram — for a TCP probe that's enough to carry both
// ports. ok is false if the quote is too short or isn't TCP.
func quotedTCPPorts(data []byte) (src, dst uint16, ok bool) {
	if len(data) < 20 {
		return 0, 0, false
	}
	ihl := int(data[0]&0x0f) * 4
	if ihl < 20 || ihl+4 > len(data) {
		return 0, 0, false
	}
	if data[9] != 6 { // IP protocol field: 6 = TCP
		return 0, 0, false
	}
	src = binary.BigEndian.Uint16(data[ihl : ihl+2])
	dst = binary.BigEndian.Uint16(data[ihl+2 : ihl+4])
	return src, dst, true
}
