package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/hriday/meridian/internal/arpscan"
	"github.com/hriday/meridian/internal/ipspace"
	"github.com/hriday/meridian/internal/lanrecon"
)

// runReconcile sweeps the local subnet with ARP and diffs the live hosts against
// the saved IP-space inventory: untracked hosts, MAC changes (possible rogues),
// and active records that no longer answer. With the apply flags it writes the
// chosen changes back and saves the store.
func runReconcile(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ipspace reconcile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", ipspace.DefaultPath(), "store file path")
	pass := fs.String("passphrase", "", "store passphrase (or MERIDIAN_PASSPHRASE)")
	window := fs.Duration("wait", arpscan.DefaultWindow, "how long to listen for ARP replies")
	max := fs.Int("max", arpscan.DefaultMaxHosts, "maximum hosts to probe")
	noVendor := fs.Bool("no-vendor", false, "skip the online MAC-vendor lookup")
	doImport := fs.Bool("import", false, "import new (untracked) hosts into the inventory")
	updateMAC := fs.Bool("update-mac", false, "update records whose MAC changed")
	deprecate := fs.Bool("deprecate-missing", false, "mark unanswered active records deprecated")
	asJSON := fs.Bool("json", false, "output as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}

	passphrase := *pass
	if passphrase == "" {
		passphrase = os.Getenv("MERIDIAN_PASSPHRASE")
	}

	var store *ipspace.Store
	if ipspace.Exists(*file) {
		events, err := ipspace.Load(*file, passphrase)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		store = ipspace.FromEvents(ipspace.Actor(), events)
	} else {
		// No inventory yet — start empty so a first sweep can bootstrap one.
		store = ipspace.NewStore(ipspace.Actor())
	}

	if !arpscan.Supported() {
		fmt.Fprintln(stderr, "ARP scanning isn't supported on this operating system yet")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Fprintln(stderr, "Sweeping the local subnet with ARP…")
	res, err := arpscan.Run(ctx, *window, *max)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !*noVendor {
		arpscan.LookupVendors(ctx, &http.Client{Timeout: 8 * time.Second}, res.Hosts)
	}

	report := lanrecon.Reconcile(res, store.Records())

	opts := lanrecon.ApplyOptions{ImportNew: *doImport, UpdateMAC: *updateMAC, DeprecateMissing: *deprecate}
	var applied lanrecon.ApplyResult
	if opts.ImportNew || opts.UpdateMAC || opts.DeprecateMissing {
		applied, err = lanrecon.Apply(store, report.Findings, opts)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := ipspace.Save(*file, passphrase, store.Events()); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	if *asJSON {
		return writeJSON(stdout, stderr, struct {
			lanrecon.Report
			Applied lanrecon.ApplyResult `json:"applied"`
		}{report, applied})
	}
	printReconcile(stdout, report, applied)
	return 0
}

func printReconcile(w io.Writer, r lanrecon.Report, a lanrecon.ApplyResult) {
	fmt.Fprintf(w, "Reconciled %s — %d new, %d mac-change, %d missing, %d in-sync\n\n",
		r.Subnet, r.New, r.MACChange, r.Missing, r.InSync)

	group := func(title string, cat lanrecon.Category) {
		var rows []lanrecon.Finding
		for _, f := range r.Findings {
			if f.Category == cat {
				rows = append(rows, f)
			}
		}
		if len(rows) == 0 {
			return
		}
		fmt.Fprintf(w, "%s:\n", title)
		for _, f := range rows {
			switch cat {
			case lanrecon.MACChange:
				fmt.Fprintf(w, "  %-16s %-18s was %s  %s\n", f.IP, f.MAC, f.KnownMAC, f.Name)
			case lanrecon.Missing:
				fmt.Fprintf(w, "  %-16s %-18s %s (didn't answer)\n", f.IP, f.MAC, f.Name)
			default:
				fmt.Fprintf(w, "  %-16s %-18s %s\n", f.IP, f.MAC, f.Vendor)
			}
		}
		fmt.Fprintln(w)
	}
	group("New (untracked)", lanrecon.New)
	group("MAC changed (possible rogue)", lanrecon.MACChange)
	group("Missing (in inventory, didn't answer)", lanrecon.Missing)

	if a.Imported > 0 || a.Updated > 0 || a.Deprecated > 0 {
		fmt.Fprintf(w, "Applied: %d imported, %d updated, %d deprecated.\n", a.Imported, a.Updated, a.Deprecated)
	} else if r.New > 0 {
		fmt.Fprintln(w, "Re-run with --import to add the new hosts to your inventory.")
	}
}
