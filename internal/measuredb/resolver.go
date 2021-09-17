package measuredb

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapResolver wraps a Resolver to add measuredb capabilities.
func WrapResolver(db DB, r netxlite.Resolver) netxlite.Resolver {
	return &resolverDB{Resolver: r, DB: db}
}

type resolverDB struct {
	netxlite.Resolver
	DB
}

// LookupHost contains the result of a host lookup.
type LookupHost struct {
	RoundTripID int64     // HTTP round trip ID
	Network     string    // network used by the resolver (e.g., "dot")
	Address     string    // address of the resolver (e.g., "8.8.4.4:853")
	Domain      string    // domain to resolve
	Started     time.Time // when we started
	Finished    time.Time // when we finished
	Error       error     // error or nil
	Addrs       []string  // resolved addrs or nil
}

func (r *resolverDB) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Now()
	r.DB.InsertIntoLookupHost(&LookupHost{
		RoundTripID: r.DB.HTTPRoundTripID(),
		Network:     r.Resolver.Network(),
		Address:     r.Resolver.Address(),
		Domain:      domain,
		Started:     started,
		Finished:    finished,
		Error:       err,
		Addrs:       addrs,
	})
	return addrs, err
}
