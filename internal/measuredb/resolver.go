package measuredb

// This file contains code to wrap netxlite.Resolver instances
// to add support for measuredb measurements

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// WrapResolver wraps a Resolver to add measuredb capabilities.
//
// LookupHost algorithm
//
// Very simple: perform the operation, store the corresponding
// event into the database, return to the caller.
func WrapResolver(db DB, r netxlite.Resolver) netxlite.Resolver {
	return &resolverDB{Resolver: r, db: db}
}

type resolverDB struct {
	netxlite.Resolver
	db DB
}

// LookupHostEvent contains the result of a host lookup.
//
// HTTPRoundTripID only make sense when we are using
// precise HTTP round trip measurements.
type LookupHostEvent struct {
	HTTPRoundTripID int64     // HTTP round trip ID
	Network         string    // network used by the resolver (e.g., "dot")
	Address         string    // address of the resolver (e.g., "8.8.4.4:853")
	Domain          string    // domain to resolve
	Started         time.Time // when we started
	Finished        time.Time // when we finished
	Error           error     // error or nil
	Addrs           []string  // resolved addrs or nil
}

func (r *resolverDB) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Now()
	r.db.InsertIntoLookupHost(&LookupHostEvent{
		HTTPRoundTripID: r.db.HTTPRoundTripID(),
		Network:         r.Resolver.Network(),
		Address:         r.Resolver.Address(),
		Domain:          domain,
		Started:         started,
		Finished:        finished,
		Error:           err,
		Addrs:           addrs,
	})
	return addrs, err
}

// WrapResolvers creates a resolver with measuredb capabilities
// out of a list of one or more resolvers.
//
// CAVEAT: passing no resolvers as parameters create a new
// resolver that always returns NXDOMAIN for every query.
//
// WrapResolvers algorithm
//
// - wrap every element in `or` using WrapResolver
//
// - return a compound resolver type (unexported)
//
// LookupHost algorithm
//
// - try every child resolver
//
// - return union of returned addrs or ErrOODNSNoSuchHost
//
// LookupHostWithoutRetry algorithm
//
// - always return the ErrNoDNSTransport error
//
// LookupHTTPSWithoutRetry algorithm
//
// - scan the list of underlying resolvers
//
// - if we find a "dot" or "https" resolver issue the
// HTTPS query using it
//
// - on success return the result, on failure continue looping
//
// - if we scan the whole list return ErrNoDNSTransport.
func WrapResolvers(db DB, or ...netxlite.Resolver) netxlite.Resolver {
	var wr []netxlite.Resolver
	for _, r := range or {
		wr = append(wr, WrapResolver(db, r))
	}
	return &compoundResolver{children: wr}
}

type compoundResolver struct {
	children []netxlite.Resolver
}

func (r *compoundResolver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	m := make(map[string]int)
	for _, ir := range r.children {
		addrs, err := ir.LookupHost(ctx, domain)
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			m[addr]++
		}
	}
	var out []string
	for k := range m {
		out = append(out, k)
	}
	if len(out) < 1 {
		return nil, errorsx.ErrOODNSNoSuchHost
	}
	return out, nil
}

func (r *compoundResolver) Network() string {
	return "compound"
}

func (r *compoundResolver) Address() string {
	return ""
}

func (r *compoundResolver) CloseIdleConnections() {
	for _, ir := range r.children {
		ir.CloseIdleConnections()
	}
}

func (r *compoundResolver) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	return nil, netxlite.ErrNoDNSTransport
}

func (r *compoundResolver) LookupHTTPSWithoutRetry(
	ctx context.Context, domain string) (netxlite.HTTPS, error) {
	for _, r := range r.children {
		switch r.Network() {
		// Rationale: only use encrypted transports.
		case "https", "dot":
			if https, err := r.LookupHTTPSWithoutRetry(ctx, domain); err == nil {
				return https, nil
			}
		}
	}
	return nil, netxlite.ErrNoDNSTransport
}
