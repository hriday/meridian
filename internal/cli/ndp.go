package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/ndp"
)

func runNDP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ndp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	withRA := fs.Bool("ra", false, "also solicit and show Router Advertisements (needs sudo)")
	iface := fs.String("iface", "", "restrict RA solicitation to this interface")
	wait := fs.Duration("wait", 4*time.Second, "how long to listen for Router Advertisements")
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *wait+3*time.Second)
	defer cancel()

	res := ndp.Result{OS: runtime.GOOS}
	ns, err := ndp.Neighbors(ctx)
	if err != nil {
		fmt.Fprintln(stderr, "couldn't read the IPv6 neighbor cache:", err)
		return 1
	}
	res.Neighbors = ns
	res.DefaultRouters, _ = ndp.DefaultRouters(ctx) // unprivileged, best-effort
	if *withRA {
		if !ndp.RASupported() {
			res.RANote = "Router Advertisement listening isn't available on this OS."
		} else if ras, rerr := ndp.ListenRA(ctx, *iface, *wait); rerr != nil {
			res.RANote = rerr.Error()
		} else {
			res.Routers = ras
		}
	}

	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printNDP(stdout, res, *withRA)
	return 0
}

func printNDP(stdout io.Writer, r ndp.Result, withRA bool) {
	fmt.Fprintf(stdout, "IPv6 neighbors (%d)\n\n", len(r.Neighbors))
	for _, n := range r.Neighbors {
		mac := n.MAC
		if mac == "" {
			mac = "(incomplete)"
		}
		flag := ""
		if n.Router {
			flag = "  router"
		}
		fmt.Fprintf(stdout, "%-42s %-18s %-8s %s%s\n", n.IP, mac, n.Interface, n.State, flag)
	}

	// Learned routers — unprivileged, from the OS's own RA-derived routing tables.
	fmt.Fprintf(stdout, "\nIPv6 routers (%d)\n\n", len(r.DefaultRouters))
	for _, rt := range r.DefaultRouters {
		addr := rt.Address
		if rt.Interface != "" {
			addr += "%" + rt.Interface
		}
		extra := ""
		if rt.Preference != "" {
			extra += "  pref=" + rt.Preference
		}
		if rt.Expire != "" {
			extra += "  expire=" + rt.Expire
		}
		fmt.Fprintf(stdout, "%-42s%s\n", addr, extra)
	}

	if !withRA {
		return
	}
	fmt.Fprintln(stdout)
	if r.RANote != "" {
		fmt.Fprintln(stdout, "Router Advertisements:", r.RANote)
		return
	}
	if len(r.Routers) == 0 {
		fmt.Fprintln(stdout, "Router Advertisements (0) — none captured in the window. Routers also send them periodically; try a longer --wait, or rely on the unprivileged routers above.")
		return
	}
	fmt.Fprintf(stdout, "Router Advertisements (%d)\n", len(r.Routers))
	for _, ra := range r.Routers {
		fmt.Fprintf(stdout, "\n%s  on %s  (lifetime %ds, pref/hops %d)\n", ra.Router, ra.Interface, ra.RouterLifetime, ra.HopLimit)
		fmt.Fprintf(stdout, "  flags   M=%s O=%s\n", yn(ra.Managed), yn(ra.OtherConfig))
		if ra.MTU > 0 {
			fmt.Fprintf(stdout, "  mtu     %d\n", ra.MTU)
		}
		for _, p := range ra.Prefixes {
			fmt.Fprintf(stdout, "  prefix  %s  (%s)  valid=%ds\n", p.Prefix, prefixFlags(p), p.ValidLifetime)
		}
		if len(ra.DNS) > 0 {
			fmt.Fprintf(stdout, "  rdnss   %s\n", strings.Join(ra.DNS, ", "))
		}
		if len(ra.DNSSearch) > 0 {
			fmt.Fprintf(stdout, "  search  %s\n", strings.Join(ra.DNSSearch, ", "))
		}
	}
}

func yn(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func prefixFlags(p ndp.Prefix) string {
	var f []string
	if p.OnLink {
		f = append(f, "on-link")
	}
	if p.Autonomous {
		f = append(f, "SLAAC")
	}
	if len(f) == 0 {
		return "no flags"
	}
	return strings.Join(f, ", ")
}
