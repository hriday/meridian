package meshtastic

import (
	"strings"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
)

// RadioConfig is the node's LoRa setup, read straight from its config dump — the
// handful of settings that decide which other nodes it can even hear. Two nodes
// must agree on all of these (and be in range) to see each other.
type RadioConfig struct {
	Region         string `json:"region"`         // e.g. "US"; "" when unset
	RegionSet      bool   `json:"regionSet"`      // false = radio won't transmit
	ModemPreset    string `json:"modemPreset"`    // e.g. "LongFast"; "" when custom radio params are in use
	UsePreset      bool   `json:"usePreset"`      // false = custom bandwidth/spread/coding, not a named preset
	TxEnabled      bool   `json:"txEnabled"`      // false = listen-only, never announces itself
	PrimaryChannel string `json:"primaryChannel"` // primary channel name; "" = the default public channel (only meaningful when ChannelKnown)
	ChannelKnown   bool   `json:"channelKnown"`   // did the dump actually carry the primary channel? (false = don't claim a channel)
}

// Finding is one mesh-health call-out: a likely reason the node is isolated, or
// a setting worth checking, each with a concrete fix. Level separates what
// Meridian can prove from what it can't:
//
//	"problem" — something on THIS node that blocks contact and you can fix
//	            (region unset, transmit disabled). Meridian sees these for certain.
//	"check"   — something Meridian can't see from here (the peer's settings,
//	            range, whether anyone else is on). Surfaced so you can verify it.
type Finding struct {
	Level string `json:"level"`
	Title string `json:"title"`
	Fix   string `json:"fix,omitempty"`
}

// radioFromLoRa builds a RadioConfig from the node's LoRaConfig and the name of
// its primary channel. Returns nil when the dump carried no LoRa config, so the
// caller can say "couldn't read the radio config" rather than invent one.
func radioFromLoRa(l *pb.Config_LoRaConfig, primaryChannel string, channelKnown bool) *RadioConfig {
	if l == nil {
		return nil
	}
	region := l.GetRegion()
	return &RadioConfig{
		Region:         regionName(region),
		RegionSet:      region != pb.Config_LoRaConfig_UNSET,
		ModemPreset:    presetName(l),
		UsePreset:      l.GetUsePreset(),
		TxEnabled:      l.GetTxEnabled(),
		PrimaryChannel: primaryChannel,
		ChannelKnown:   channelKnown,
	}
}

// diagnose turns a snapshot into mesh-health findings, most important first. It
// is pure, so the whole "why can't I see nodes" logic is testable without a
// radio. It never claims the node is fine — when the config looks capable but
// the node is alone, it says what a peer must match and that range is unknown.
func diagnose(s Snapshot) []Finding {
	var fs []Finding
	r := s.Radio

	// Problems on this node that block all contact — Meridian sees these for sure.
	if r != nil {
		if !r.RegionSet {
			fs = append(fs, Finding{
				Level: "problem",
				Title: "LoRa region is not set — the radio will not transmit.",
				Fix:   "Set your region on the node (Settings → LoRa → Region) to match where you are. Until it's set, the device stays off the air and can't reach any mesh.",
			})
		}
		if !r.TxEnabled {
			fs = append(fs, Finding{
				Level: "problem",
				Title: "Transmit is disabled — this node can listen but never announces itself.",
				Fix:   "Enable TX in the node's LoRa config so neighbours can hear it and it shows up in their node lists.",
			})
		}
	}

	// Isolation: the node sees only itself.
	if len(s.Nodes) == 0 {
		switch {
		case r == nil:
			fs = append(fs, Finding{
				Level: "check",
				Title: "This node sees no other nodes.",
				Fix:   "Confirm its region, modem preset and channel match a peer that's powered on and in range.",
			})
		case r.RegionSet && r.TxEnabled:
			// The config can talk; isolation is beyond what Meridian can see.
			// Only lean on the airtime reading when the node actually reported
			// metrics — absent metrics default to 0 and must not become a verdict.
			// round1 means a light-but-real 0.04% also reads as ~0, so the wording
			// stays "little or no", never "none".
			quiet := s.metricsKnown && s.ChannelUtil == 0 && s.AirUtilTx == 0
			title := "On the air, but no neighbours yet — nothing Meridian can see is wrong with this node."
			if quiet {
				title = "No neighbours, and this node's channel utilisation reads about 0%."
			}
			fix := "A peer has to match your region (" + r.Region + "), modem preset (" + presetLabel(r) + ") and primary channel (" + channelLabel(r) + "), be powered on, and be within LoRa range."
			if quiet {
				fix = "Little or no traffic is reaching this node, which points more to range or antenna than to a settings mismatch. " + fix
			}
			fs = append(fs, Finding{Level: "check", Title: title, Fix: fix})
			// region-unset / tx-disabled cases are already explained by the problem findings above.
		}
	}

	return fs
}

// presetLabel / channelLabel render the comparison values for a "check" finding.
func presetLabel(r *RadioConfig) string {
	if r.ModemPreset == "" {
		return "custom radio settings"
	}
	return r.ModemPreset
}

func channelLabel(r *RadioConfig) string {
	if !r.ChannelKnown {
		return "its channel (not in this read)"
	}
	if strings.TrimSpace(r.PrimaryChannel) == "" {
		return "the default public channel"
	}
	return r.PrimaryChannel
}

// regionName is the region code as a short label ("US", "EU_868"), or "" unset.
func regionName(code pb.Config_LoRaConfig_RegionCode) string {
	if code == pb.Config_LoRaConfig_UNSET {
		return ""
	}
	return code.String()
}

// presetName renders the modem preset the Meshtastic way ("LongFast"), or "" when
// the node uses custom radio parameters instead of a named preset.
func presetName(l *pb.Config_LoRaConfig) string {
	if !l.GetUsePreset() {
		return ""
	}
	return camelWords(l.GetModemPreset().String())
}

// camelWords turns an UPPER_SNAKE enum name into Meshtastic's CamelCase display
// form: "LONG_FAST" -> "LongFast", "VERY_LONG_SLOW" -> "VeryLongSlow".
func camelWords(s string) string {
	var b strings.Builder
	for _, part := range strings.Split(strings.ToLower(s), "_") {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}
