package dnsprop

// Public returns a spread of well-known public DNS resolvers across different
// operators and regions, so a propagation check samples the global view rather
// than one provider's.
func Public() []Resolver {
	return []Resolver{
		{Name: "Google", IP: "8.8.8.8", Loc: "global anycast"},
		{Name: "Google 2", IP: "8.8.4.4", Loc: "global anycast"},
		{Name: "Cloudflare", IP: "1.1.1.1", Loc: "global anycast"},
		{Name: "Cloudflare 2", IP: "1.0.0.1", Loc: "global anycast"},
		{Name: "Quad9", IP: "9.9.9.9", Loc: "global anycast"},
		{Name: "OpenDNS", IP: "208.67.222.222", Loc: "Cisco, US"},
		{Name: "AdGuard", IP: "94.140.14.14", Loc: "Cyprus/EU"},
		{Name: "Level3", IP: "4.2.2.2", Loc: "CenturyLink, US"},
		{Name: "Comodo", IP: "8.26.56.26", Loc: "US"},
		{Name: "DNS.Watch", IP: "84.200.69.80", Loc: "Germany"},
		{Name: "Yandex", IP: "77.88.8.8", Loc: "Russia"},
		{Name: "CleanBrowsing", IP: "185.228.168.9", Loc: "US/EU"},
	}
}
