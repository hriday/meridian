#import <CoreLocation/CoreLocation.h>
#import "location_darwin.h"

// A single retained manager: CLLocationManager must stay alive for the
// authorization prompt to appear and for the status to be queryable.
static CLLocationManager *gManager = nil;

static CLLocationManager *manager(void) {
    if (gManager == nil) {
        gManager = [[CLLocationManager alloc] init];
    }
    return gManager;
}

const char *meridian_location_status(void) {
    CLAuthorizationStatus st;
    if (@available(macOS 11.0, *)) {
        st = [manager() authorizationStatus];
    } else {
        st = [CLLocationManager authorizationStatus];
    }
    switch (st) {
        case kCLAuthorizationStatusNotDetermined:    return "notDetermined";
        case kCLAuthorizationStatusRestricted:       return "restricted";
        case kCLAuthorizationStatusDenied:           return "denied";
        case kCLAuthorizationStatusAuthorizedAlways: return "authorized"; // macOS grant maps here
        default:                                     return "unknown";
    }
}

void meridian_request_location(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        CLLocationManager *m = manager();
        [m requestWhenInUseAuthorization];
        // A brief location start nudges macOS to register an actual use by this
        // app, which is what unlocks SSID visibility via airportd. We don't need
        // coordinates, so no delegate is required.
        [m startUpdatingLocation];
    });
}
