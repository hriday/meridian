package app

import "github.com/hriday/meridian/internal/webconf"

// AnalyzeWebServer parses a web-server config export (an `nginx -T` dump or a
// single file) offline and returns the structure, routing, and findings.
// server defaults to "nginx".
func (a *App) AnalyzeWebServer(configText, server string) (webconf.Analysis, error) {
	return webconf.Analyze(configText, server)
}
