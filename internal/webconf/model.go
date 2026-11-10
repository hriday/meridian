package webconf

// Model types. nginx.go fills Servers/Upstreams; findings.go fills Findings and
// Counts.

// Listener is one `listen` directive on a server block.
type Listener struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	SSL     bool   `json:"ssl"`
	HTTP2   bool   `json:"http2"`
	Default bool   `json:"default"`
}

// Location is one `location` block and where it routes.
type Location struct {
	Path   string   `json:"path"`
	Kind   string   `json:"kind"`   // static | proxy | redirect | alias
	Target string   `json:"target"` // upstream name | return target | alias path
	Flags  []string `json:"flags"`
}

// Server is one `server` block (virtual host).
type Server struct {
	Names        []string   `json:"names"`
	Listeners    []Listener `json:"listeners"`
	Root         string     `json:"root"`
	Locations    []Location `json:"locations"`
	TLSProtocols []string   `json:"tlsProtocols"`
	TLSCert      string     `json:"tlsCert"`
	Access       []string   `json:"access"`
	ServerTokens string     `json:"serverTokens"`
	Headers      []string   `json:"headers"`
	File         string     `json:"file"`
	Line         int        `json:"line"`
}

// Upstream is one `upstream` block (backend pool).
type Upstream struct {
	Name    string   `json:"name"`
	Servers []string `json:"servers"`
	Policy  string   `json:"policy"`
	UsedBy  []string `json:"usedBy"`
	File    string   `json:"file"`
	Line    int      `json:"line"`
}

// Finding is one problem or note.
type Finding struct {
	Severity string `json:"severity"` // error | warning | info
	Category string `json:"category"` // breaking | security | best-practice
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Counts are the headline numbers.
type Counts struct {
	Servers   int `json:"servers"`
	Listeners int `json:"listeners"`
	Upstreams int `json:"upstreams"`
	Locations int `json:"locations"`
	Errors    int `json:"errors"`
	Warnings  int `json:"warnings"`
}

// Analysis is the result for one config.
type Analysis struct {
	ServerType string     `json:"serverType"`
	Servers    []Server   `json:"servers"`
	Upstreams  []Upstream `json:"upstreams"`
	Findings   []Finding  `json:"findings"`
	Counts     Counts     `json:"counts"`
	Notes      []string   `json:"notes"`

	// http-level context for inheritance-aware findings; never serialized.
	httpTokens  string
	httpHeaders []string
}

func newAnalysis() Analysis {
	return Analysis{
		ServerType: "nginx",
		Servers:    []Server{},
		Upstreams:  []Upstream{},
		Findings:   []Finding{},
		Notes:      []string{},
	}
}
