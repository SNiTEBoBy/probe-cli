package measuredb

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// LookupHTTPSEvent is the event emitted when we perform
// an HTTPS/SVCB DNS query for a domain.
//
// CAVEAT: HTTPRoundTripID is reliable only when the
// underlying DB supports precise round trip measurements.
type LookupHTTPSEvent struct {
	HTTPRoundTripID int64     // HTTP round trip ID
	Domain          string    // domain to resolve
	Started         time.Time // when we started
	Finished        time.Time // when we finished
	Error           error     // error or nil
	IPv4            []string  // resolved addrs or nil
	IPv6            []string  // resolved addrs or nil
	ALPN            []string  // ALPNs or nil
}

func svcbLookupHTTPSWithoutRetry(ctx context.Context,
	db DB, reso netxlite.Resolver, domain string) (netxlite.HTTPS, error) {
	started := time.Now()
	https, err := reso.LookupHTTPSWithoutRetry(ctx, domain)
	finished := time.Now()
	ev := &LookupHTTPSEvent{
		HTTPRoundTripID: db.HTTPRoundTripID(),
		Domain:          domain,
		Started:         started,
		Finished:        finished,
		Error:           err,
	}
	if err == nil {
		ev.IPv4 = https.IPv4Hint()
		ev.IPv6 = https.IPv6Hint()
		ev.ALPN = https.ALPN()
	}
	db.InsertIntoLookupHTTPS(ev)
	return https, err
}
