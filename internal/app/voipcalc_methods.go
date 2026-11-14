package app

import "github.com/hriday/meridian/internal/voipcalc"

// VoipCodecs lists the supported codecs for the bandwidth calculator.
func (a *App) VoipCodecs() []voipcalc.Codec { return voipcalc.Codecs() }

// VoipBandwidth computes per-stream / per-call / total VoIP bandwidth for a
// codec, packet interval, and number of concurrent calls.
func (a *App) VoipBandwidth(codec string, ptimeMs, calls int) (voipcalc.Result, error) {
	return voipcalc.Calc(codec, ptimeMs, calls)
}
