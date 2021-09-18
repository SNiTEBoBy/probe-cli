package measuredb

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// WrapDNSRoundTripper wraps a DNS round tripper adding measuredb capabilities.
func WrapDNSRoundTripper(db DB, rt dnsx.RoundTripper) dnsx.RoundTripper {
	return &dnsTransportDB{DB: db, RoundTripper: rt}
}

type dnsTransportDB struct {
	dnsx.RoundTripper
	DB
}

// TODO(bassosimone): we should rename RoundTripID to
// HTTPRoundTripID otherwise there will be confusion
// between the HTTP and the DNS round trip.

// DNSRoundTrip contains the result of a DNS round trip.
//
// RoundTripID only make sense when we are using precise
// HTTP round trip measurements.
type DNSRoundTrip struct {
	RoundTripID int64
	Network     string
	Address     string
	Query       []byte
	Started     time.Time
	Finished    time.Time
	Error       error
	Reply       []byte
}

func (txp *dnsTransportDB) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Now()
	reply, err := txp.RoundTripper.RoundTrip(ctx, query)
	finished := time.Now()
	txp.DB.InsertIntoDNSRoundTrip(&DNSRoundTrip{
		RoundTripID: txp.DB.HTTPRoundTripID(),
		Network:     txp.RoundTripper.Network(),
		Address:     txp.RoundTripper.Address(),
		Query:       query,
		Started:     started,
		Finished:    finished,
		Error:       err,
		Reply:       reply,
	})
	return reply, err
}
