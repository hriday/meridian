#import <CoreLocation/CoreLocation.h>
#import <CoreWLAN/CoreWLAN.h>
#import <Foundation/Foundation.h>
#import <string.h>
#import "location_darwin.h"

// The authorization status is cached here and kept current by the delegate.
// A CLLocationManager only reports an accurate status once it has been created
// on a thread with a live run loop and its delegate has been called — querying
// a fresh manager off the main thread returns "notDetermined" even when the
// grant exists, which is the bug this replaces.
static volatile CLAuthorizationStatus gStatus = kCLAuthorizationStatusNotDetermined;

@interface MeridianLocDelegate : NSObject <CLLocationManagerDelegate>
@end

@implementation MeridianLocDelegate
- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)m {
    if (@available(macOS 11.0, *)) {
        gStatus = m.authorizationStatus;
    }
}
@end

static CLLocationManager *gManager = nil;
static MeridianLocDelegate *gDelegate = nil;

// runOnMain runs block synchronously on the main queue (where CLLocationManager
// needs to live), unless already on the main thread.
static void runOnMain(void (^block)(void)) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_sync(dispatch_get_main_queue(), block);
    }
}

// ensureManager creates the shared manager on the main thread with a delegate
// and seeds the cached status, exactly once.
static void ensureManager(void) {
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        runOnMain(^{
            gDelegate = [[MeridianLocDelegate alloc] init];
            gManager = [[CLLocationManager alloc] init];
            gManager.delegate = gDelegate;
            if (@available(macOS 11.0, *)) {
                gStatus = gManager.authorizationStatus; // seed before the delegate fires
            }
        });
    });
}

const char *meridian_location_status(void) {
    ensureManager();
    switch (gStatus) {
        case kCLAuthorizationStatusNotDetermined:    return "notDetermined";
        case kCLAuthorizationStatusRestricted:       return "restricted";
        case kCLAuthorizationStatusDenied:           return "denied";
        case kCLAuthorizationStatusAuthorizedAlways: return "authorized"; // macOS grant maps here
        default:                                     return "unknown";
    }
}

void meridian_request_location(void) {
    ensureManager();
    runOnMain(^{
        [gManager requestWhenInUseAuthorization];
        // A brief location start nudges macOS to register an actual use by this
        // app, which is what unlocks SSID visibility via airportd. We don't need
        // coordinates, so no delegate callback for location is required.
        [gManager startUpdatingLocation];
    });
}

char *meridian_current_ssid(void) {
    @autoreleasepool {
        CWInterface *iface = [[CWWiFiClient sharedWiFiClient] interface];
        NSString *ssid = [iface ssid];
        if (ssid == nil || ssid.length == 0) {
            return NULL;
        }
        return strdup([ssid UTF8String]);
    }
}

static NSString *bandString(CWChannelBand b) {
    switch (b) {
        case kCWChannelBand2GHz: return @"2GHz";
        case kCWChannelBand5GHz: return @"5GHz";
        default:
            if (@available(macOS 12.0, *)) {
                if (b == kCWChannelBand6GHz) return @"6GHz";
            }
            return @"";
    }
}

static NSString *widthString(CWChannelWidth w) {
    switch (w) {
        case kCWChannelWidth20MHz:  return @"20MHz";
        case kCWChannelWidth40MHz:  return @"40MHz";
        case kCWChannelWidth80MHz:  return @"80MHz";
        case kCWChannelWidth160MHz: return @"160MHz";
        default:                    return @"";
    }
}

char *meridian_scan_networks(void) {
    @autoreleasepool {
        CWInterface *iface = [[CWWiFiClient sharedWiFiClient] interface];
        if (iface == nil) {
            return NULL;
        }
        NSError *err = nil;
        NSSet<CWNetwork *> *nets = [iface scanForNetworksWithSSID:nil error:&err];
        if (nets == nil) {
            return NULL;
        }
        NSString *connBSSID = [iface bssid];
        NSMutableArray *arr = [NSMutableArray array];
        for (CWNetwork *n in nets) {
            NSString *bssid = n.bssid ? n.bssid : @"";
            // Exclude the connected AP (it is reported separately as "current").
            if (connBSSID.length > 0 && bssid.length > 0 &&
                [bssid caseInsensitiveCompare:connBSSID] == NSOrderedSame) {
                continue;
            }
            CWChannel *ch = n.wlanChannel;
            NSString *ssid = n.ssid ? n.ssid : @"";
            [arr addObject:@{
                @"ssid":    ssid,
                @"bssid":   bssid,
                @"channel": @(ch ? (NSInteger)ch.channelNumber : 0),
                @"band":    ch ? bandString(ch.channelBand) : @"",
                @"width":   ch ? widthString(ch.channelWidth) : @"",
                @"rssi":    @((NSInteger)n.rssiValue),
            }];
        }
        NSData *json = [NSJSONSerialization dataWithJSONObject:arr options:0 error:&err];
        if (json == nil) {
            return NULL;
        }
        NSString *s = [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding];
        if (s == nil) {
            return NULL;
        }
        return strdup([s UTF8String]);
    }
}
