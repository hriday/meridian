package meshtastic

import (
	"strings"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
)

// hasLetter reports whether s contains a letter — a protobuf enum String()
// returns a bare decimal for an unknown value, which we'd rather drop than show.
func hasLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

// hwModel renders a hardware-model enum as its canonical token (e.g. "HELTEC_V3"),
// dropping the UNSET sentinel and any unknown numeric value.
func hwModel(m pb.HardwareModel) string {
	s := m.String()
	if s == "UNSET" || !hasLetter(s) {
		return ""
	}
	return s
}

// role renders a device-role enum in title case (e.g. CLIENT_MUTE -> "Client Mute").
func role(r pb.Config_DeviceConfig_Role) string {
	s := r.String()
	if !hasLetter(s) {
		return ""
	}
	parts := strings.Split(strings.ToLower(s), "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
