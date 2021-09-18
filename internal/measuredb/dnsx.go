package measuredb

// This file contains code to wrap netxlite/dnsx (i.e., the DNS
// extensions, to add measuredb capabilities)

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// WrapDNSRoundTripper wraps a DNS round tripper adding measuredb capabilities.
//
// The RoundTrip algorithm is very simple: it will run the DNS round trip, store
// the results into the database, then return to the caller.
func WrapDNSRoundTripper(db DB, rt dnsx.RoundTripper) dnsx.RoundTripper {
	return &dnsTransportDB{DB: db, RoundTripper: rt}
}

type dnsTransportDB struct {
	dnsx.RoundTripper
	DB
}

// DNSRoundTripEvent contains the result of a DNS round trip.
//
// HTTPRoundTripID only make sense when we are using precise
// HTTP round trip measurements.
type DNSRoundTripEvent struct {
	HTTPRoundTripID int64
	Network         string
	Address         string
	Query           []byte
	Started         time.Time
	Finished        time.Time
	Error           error
	Reply           []byte
}

func (txp *dnsTransportDB) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Now()
	reply, err := txp.RoundTripper.RoundTrip(ctx, query)
	finished := time.Now()
	txp.DB.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		HTTPRoundTripID: txp.DB.HTTPRoundTripID(),
		Network:         txp.RoundTripper.Network(),
		Address:         txp.RoundTripper.Address(),
		Query:           query,
		Started:         started,
		Finished:        finished,
		Error:           err,
		Reply:           reply,
	})
	return reply, err
}
