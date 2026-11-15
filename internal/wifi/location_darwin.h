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

#endif
