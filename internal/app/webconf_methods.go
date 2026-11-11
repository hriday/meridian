package app

import "github.com/hriday/meridian/internal/webconf"

// AnalyzeWebServer parses a web-server config offline and returns the
// structure, routing, and findings. server is one of "nginx" (an `nginx -T`
// dump or a single file), "apache" (httpd.conf), or "caddy" (a Caddyfile);
// defaults to "nginx".
func (a *App) AnalyzeWebServer(configText, server string) (webconf.Analysis, error) {
	return webconf.Analyze(configText, server)
}
