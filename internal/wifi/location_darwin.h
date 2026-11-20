#ifndef MERIDIAN_LOCATION_H
#define MERIDIAN_LOCATION_H

// meridian_location_status returns the app's Core Location authorization as a
// static C string: "notDetermined", "denied", "restricted", "authorized" or
// "unknown". Safe to call from any thread.
const char *meridian_location_status(void);

// meridian_request_location asks macOS for Location Services authorization
// (When-In-Use). This is what makes Meridian appear in System Settings ›
// Privacy & Security › Location Services and, once granted, lets airportd
// reveal Wi-Fi SSIDs. It is dispatched to the main queue and returns at once;
// the system prompt is shown asynchronously.
void meridian_request_location(void);

// meridian_current_ssid returns the connected network's SSID read via CoreWLAN
// in THIS process — so the app's own Location grant applies, unlike the
// separate `system_profiler` process which can still redact it. Returns NULL
// when not connected or not authorized. The caller frees the result with free().
char *meridian_current_ssid(void);

// meridian_scan_networks runs an in-process CoreWLAN active scan and returns the
// neighboring APs (the connected AP excluded) as a JSON array of objects with
// keys ssid, bssid, channel, band, width, rssi. Because the scan runs in THIS
// process, the app's Location grant applies and SSIDs are real; without the
// grant macOS returns the networks with empty ssids (the Go side drops those
// and falls back to system_profiler). Returns NULL on error. Caller frees with
// free().
char *meridian_scan_networks(void);

#endif
