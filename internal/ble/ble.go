// Package ble takes a point-in-time survey of the Bluetooth Low Energy devices
// advertising nearby: it scans for a few seconds, aggregates the advertisement
// packets by device, and reports each device with its signal (RSSI), name and
// advertised company. It keeps no history — scan, read the room, move on.
//
// Scanning uses the host's own BLE radio in-process (CoreBluetooth on macOS,
// BlueZ on Linux, WinRT on Windows via tinygo.org/x/bluetooth). On macOS the
// addresses are per-host CoreBluetooth UUIDs, not hardware MACs (Apple privacy).
package ble

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// Survey is one point-in-time read of the nearby BLE devices.
type Survey struct {
	DurationSec int      `json:"durationSec"` // how long the scan ran
	Devices     []Device `json:"devices"`     // aggregated, strongest signal first
	Count       int      `json:"count"`       // distinct devices seen
	Named       int      `json:"named"`       // how many advertised a name
	TakenAt     string   `json:"takenAt"`     // RFC3339 UTC instant of this survey
}

// Device is one nearby BLE device, folded from all its advertisement packets.
type Device struct {
	Address string `json:"address"` // CoreBluetooth UUID (macOS) or MAC (Linux/Windows)
	Name    string `json:"name"`    // advertised local name, if any
	RSSI    int    `json:"rssi"`    // strongest signal seen, dBm (closer to 0 = nearer)
	Company string `json:"company"` // advertised manufacturer, or "0xNNNN" if the ID is unknown
	Seen    int    `json:"seen"`    // advertisement packets observed
}

// observation is one advertisement packet — the input to the pure aggregator.
type observation struct {
	address    string
	name       string
	rssi       int
	company    uint16
	hasCompany bool
}

// defaultDuration is a sensible scan window: long enough to catch devices that
// advertise every second or two, short enough to stay "point in time".
const defaultDuration = 6 * time.Second

// Scan surveys nearby BLE devices for dur (or defaultDuration if dur <= 0),
// stopping early if ctx is cancelled.
func Scan(ctx context.Context, dur time.Duration) (Survey, error) {
	if dur <= 0 {
		dur = defaultDuration
	}
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return Survey{}, fmt.Errorf("ble: couldn't start the Bluetooth adapter (%w) — check Bluetooth is on and that Meridian is allowed under System Settings › Privacy & Security › Bluetooth", err)
	}

	var (
		mu  sync.Mutex
		obs []observation
	)

	// Stop the (blocking) scan after the window, and also if ctx is cancelled.
	timer := time.AfterFunc(dur, func() { _ = adapter.StopScan() })
	defer timer.Stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = adapter.StopScan()
		case <-done:
		}
	}()

	err := adapter.Scan(func(_ *bluetooth.Adapter, r bluetooth.ScanResult) {
		o := observation{address: r.Address.String(), name: r.LocalName(), rssi: int(r.RSSI)}
		if md := r.ManufacturerData(); len(md) > 0 {
			o.company, o.hasCompany = md[0].CompanyID, true
		}
		mu.Lock()
		obs = append(obs, o)
		mu.Unlock()
	})
	if err != nil {
		return Survey{}, fmt.Errorf("ble: scan failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Survey{}, err
	}

	mu.Lock()
	defer mu.Unlock()
	return aggregate(obs, dur), nil
}

// aggregate folds advertisement packets into a Survey: one entry per device,
// keeping the strongest RSSI and the first non-empty name/company seen, then
// ranks the devices strongest-signal first. It is pure, so the folding and
// ranking are testable without a radio.
func aggregate(obs []observation, dur time.Duration) Survey {
	byAddr := map[string]*Device{}
	var order []string
	for _, o := range obs {
		d := byAddr[o.address]
		if d == nil {
			d = &Device{Address: o.address, RSSI: o.rssi}
			byAddr[o.address] = d
			order = append(order, o.address)
		}
		d.Seen++
		if o.rssi > d.RSSI {
			d.RSSI = o.rssi // strongest = closest to 0 dBm
		}
		if o.name != "" && d.Name == "" {
			d.Name = o.name
		}
		if o.hasCompany && d.Company == "" {
			d.Company = companyLabel(o.company)
		}
	}

	s := Survey{
		DurationSec: int(dur / time.Second),
		TakenAt:     time.Now().UTC().Format(time.RFC3339),
	}
	for _, a := range order {
		d := byAddr[a]
		s.Devices = append(s.Devices, *d)
		if d.Name != "" {
			s.Named++
		}
	}
	s.Count = len(s.Devices)
	rankDevices(s.Devices)
	return s
}

// rankDevices orders strongest-signal first, then named devices before unnamed
// (a named device is the more useful row), then by address for stability.
func rankDevices(ds []Device) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.RSSI != b.RSSI {
			return a.RSSI > b.RSSI
		}
		if (a.Name != "") != (b.Name != "") {
			return a.Name != ""
		}
		return a.Address < b.Address
	})
}
