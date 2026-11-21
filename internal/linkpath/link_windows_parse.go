package linkpath

import (
	"encoding/json"
	"strings"
)

// netAdapter mirrors the Get-NetAdapter fields we select.
type netAdapter struct {
	Name              string `json:"Name"`
	Speed             int64  `json:"Speed"`             // link speed, bits per second
	FullDuplex        *bool  `json:"FullDuplex"`        // null when the driver doesn't report it
	PhysicalMediaType string `json:"PhysicalMediaType"` // e.g. "802.3", "Native 802.11"
}

// parseNetAdapter reads the JSON from `Get-NetAdapter ... | ConvertTo-Json`
// (an object when one adapter matched) into a link read. Build-tag-free so it
// is unit-tested on every platform.
func parseNetAdapter(b []byte) linkState {
	var ls linkState
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 {
		return ls
	}
	var a netAdapter
	if b[0] == '[' { // Get-NetAdapter -Name is one adapter, but be defensive
		var arr []netAdapter
		if err := json.Unmarshal(b, &arr); err != nil || len(arr) == 0 {
			return ls
		}
		a = arr[0]
	} else if err := json.Unmarshal(b, &a); err != nil {
		return ls
	}

	if a.Speed > 0 {
		ls.speedMbps = int(a.Speed / 1_000_000)
	}
	if a.FullDuplex != nil {
		if *a.FullDuplex {
			ls.duplex = "full"
		} else {
			ls.duplex = "half"
		}
	}
	ls.media = a.PhysicalMediaType
	ls.wireless = strings.Contains(strings.ToLower(a.PhysicalMediaType), "802.11")
	ls.known = ls.duplex != "" || ls.speedMbps > 0
	return ls
}
