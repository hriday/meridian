package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/acmecert"
)

// CertBeginResult carries the TXT records the operator must publish.
type CertBeginResult struct {
	OrderID string               `json:"orderId"`
	Records []acmecert.DNSRecord `json:"records"`
	Staging bool                 `json:"staging"`
}

// CertResult is the issued certificate and its private key, PEM-encoded.
type CertResult struct {
	Domains   []string `json:"domains"`
	Fullchain string   `json:"fullchain"`
	Key       string   `json:"key"`
}

// CertBegin starts a Let's Encrypt DNS-01 order for the given domains and
// returns the TXT record(s) to publish. The in-flight order is held in memory
// under the returned OrderID until CertFinalize (or CertCancel). staging uses
// Let's Encrypt's staging CA (untrusted test certs, looser rate limits).
func (a *App) CertBegin(email, domainsCSV string, staging bool) (CertBeginResult, error) {
	domains := splitDomainList(domainsCSV)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	order, records, err := acmecert.Begin(ctx, acmecert.Directory(staging), strings.TrimSpace(email), domains)
	if err != nil {
		return CertBeginResult{}, err
	}

	id := newOrderID()
	a.acmeMu.Lock()
	if a.acmeOrders == nil {
		a.acmeOrders = map[string]*acmecert.Order{}
	}
	a.acmeOrders[id] = order
	a.acmeMu.Unlock()

	return CertBeginResult{OrderID: id, Records: records, Staging: staging}, nil
}

// CertFinalize tells the CA the TXT records are published, waits for validation,
// and returns the issued chain + key. The order is cleared on success.
func (a *App) CertFinalize(orderID string) (CertResult, error) {
	a.acmeMu.Lock()
	order := a.acmeOrders[orderID]
	a.acmeMu.Unlock()
	if order == nil {
		return CertResult{}, fmt.Errorf("that certificate order is no longer active — start again")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	chain, key, err := order.Finalize(ctx)
	if err != nil {
		return CertResult{}, err
	}

	a.acmeMu.Lock()
	delete(a.acmeOrders, orderID)
	a.acmeMu.Unlock()
	return CertResult{Domains: order.Domains(), Fullchain: chain, Key: key}, nil
}

// CertCancel discards an in-flight order (e.g. the user closed the tool).
func (a *App) CertCancel(orderID string) {
	a.acmeMu.Lock()
	delete(a.acmeOrders, orderID)
	a.acmeMu.Unlock()
}

func splitDomainList(csv string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(csv, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	}) {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func newOrderID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
