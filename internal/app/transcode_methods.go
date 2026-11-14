package app

import "github.com/hriday/meridian/internal/transcode"

// Transcode runs one encode/decode operation (base64, hex, URL, JSON, …) on
// input, locally.
func (a *App) Transcode(op, input string) (string, error) {
	return transcode.Apply(op, input)
}

// TranscodeDetect best-effort reports how input might be decoded.
func (a *App) TranscodeDetect(input string) []transcode.Guess {
	return transcode.Detect(input)
}

// TranscodeOps lists the supported operation names.
func (a *App) TranscodeOps() []string { return transcode.Ops() }
