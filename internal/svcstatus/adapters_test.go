package svcstatus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Apple -----------------------------------------------------------------
// Shape per the live feed: {services:[{serviceName, redirectUrl, events:[...]}]}.
// A service is down iff an event has "eventStatus":"ongoing".
const appleBody = `{"drMessage":null,"services":[
  {"serviceName":"Apple Books","redirectUrl":null,"events":[]},
  {"serviceName":"App Store Connect","redirectUrl":null,"events":[
    {"messageId":"2000005806","statusType":"Outage","message":"Investigating","datePosted":"11/17/2026 13:04 PST","startDate":"11/17/2026 11:09 PST","endDate":null,"epochStartDate":1794942540000,"epochEndDate":null,"usersAffected":"Some users are affected","affectedServices":null,"eventStatus":"ongoing"}]},
  {"serviceName":"iCloud Mail","redirectUrl":null,"events":[
    {"messageId":"1000005988","statusType":"Outage","usersAffected":"Some users were affected","startDate":"11/17/2026 08:45 PST","endDate":"11/17/2026 12:00 PST","epochEndDate":1794945600000,"eventStatus":"resolved"}]}
],"drpost":false}`

func TestAppleOngoingVsResolved(t *testing.T) {
	s, err := Check(context.Background(), &fakeDoer{body: appleBody}, "apple")
	require.NoError(t, err)
	require.Equal(t, "Apple", s.Source)
	require.False(t, s.Operational)
	require.Equal(t, "major", s.Indicator) // an Outage is major
	// Only the ongoing service is impacted; the resolved and healthy ones aren't.
	require.Len(t, s.Impacted, 1)
	require.Equal(t, "App Store Connect", s.Impacted[0].Name)
	require.Len(t, s.Incidents, 1)
}

func TestAppleJSONPDeveloperFeed(t *testing.T) {
	// The developer feed is JSONP; the parser must unwrap it.
	body := "jsonCallback(" + appleBody + ");"
	s, err := Check(context.Background(), &fakeDoer{body: body}, "apple-developer")
	require.NoError(t, err)
	require.False(t, s.Operational)
}

func TestAppleAllOperational(t *testing.T) {
	s, err := Check(context.Background(), &fakeDoer{body: `{"services":[{"serviceName":"Apple Music","events":[]}]}`}, "apple")
	require.NoError(t, err)
	require.True(t, s.Operational)
	require.Equal(t, "none", s.Indicator)
}

// --- Google ----------------------------------------------------------------
// Array of incidents; open iff `end` is empty and most_recent_update.status
// isn't AVAILABLE. Field names are from the live incidents.json.
const googleBody = `[
  {"external_desc":"Resolved region issue","begin":"2026-10-20T06:00:00+00:00","end":"2026-10-20T12:08:00+00:00","status_impact":"SERVICE_DISRUPTION","severity":"medium","service_name":"Multiple Products","affected_products":[{"title":"Compute Engine"}],"uri":"incidents/aaa","most_recent_update":{"status":"AVAILABLE","modified":"2026-10-20T12:08:00+00:00"}},
  {"external_desc":"Ongoing Spanner disruption","begin":"2026-11-17T09:00:00+00:00","end":"","status_impact":"SERVICE_DISRUPTION","severity":"high","service_name":"Cloud Spanner","affected_products":[{"title":"Cloud Spanner"}],"uri":"incidents/bbb","most_recent_update":{"status":"SERVICE_DISRUPTION","modified":"2026-11-17T10:00:00+00:00"}}
]`

func TestGoogleOpenIncidentOnly(t *testing.T) {
	s, err := Check(context.Background(), &fakeDoer{body: googleBody}, "google-cloud")
	require.NoError(t, err)
	require.Equal(t, "Google", s.Source)
	require.False(t, s.Operational)
	require.Len(t, s.Impacted, 1)
	require.Equal(t, "Cloud Spanner", s.Impacted[0].Name)
	require.Len(t, s.Incidents, 1)
	require.Equal(t, "critical", s.Incidents[0].Impact) // severity high
	require.Equal(t, "https://status.cloud.google.com/incidents/bbb", s.Incidents[0].Shortlink)
}

func TestGoogleAllResolved(t *testing.T) {
	body := `[{"external_desc":"x","end":"2026-01-01T00:00:00+00:00","most_recent_update":{"status":"AVAILABLE"}}]`
	s, err := Check(context.Background(), &fakeDoer{body: body}, "google-workspace")
	require.NoError(t, err)
	require.True(t, s.Operational)
}

// --- Azure -----------------------------------------------------------------
// RSS; healthy = zero <item>s (verified against the live feed).
const azureHealthy = `<?xml version="1.0" encoding="utf-8"?><rss version="2.0"><channel><title>Azure Status</title><lastBuildDate>Tue, 17 Nov 2026 22:06:00 Z</lastBuildDate></channel></rss>`

func TestAzureHealthyIsEmptyFeed(t *testing.T) {
	s, err := Check(context.Background(), &fakeDoer{body: azureHealthy}, "azure")
	require.NoError(t, err)
	require.Equal(t, "Azure", s.Source)
	require.True(t, s.Operational)
	require.Empty(t, s.Incidents)
}

func TestAzureItemIsActive(t *testing.T) {
	body := `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Networking - West Europe</title><link>https://azure.status.microsoft/en-us/status/history/</link><pubDate>Tue, 17 Nov 2026 20:00:00 Z</pubDate></item></channel></rss>`
	s, err := Check(context.Background(), &fakeDoer{body: body}, "azure")
	require.NoError(t, err)
	require.False(t, s.Operational)
	require.Len(t, s.Incidents, 1)
	require.Equal(t, "Networking - West Europe", s.Incidents[0].Name)
}
