package measuredb

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
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
//
// RoundTripID only make sense when we are using precise
// HTTP round trip measurements.
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

// WrapResolvers creates a compound resolver that wraps all the
// underlying resolvers for measuredb capabilities and queries all
// of them to get answers. On error it always returns NXDOMAIN.
func WrapResolvers(db DB, or ...netxlite.Resolver) netxlite.Resolver {
	var wr []netxlite.Resolver
	for _, r := range or {
		wr = append(wr, WrapResolver(db, r))
	}
	return &compoundResolver{wr: wr}
}

type compoundResolver struct {
	wr []netxlite.Resolver
}

func (r *compoundResolver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	m := make(map[string]int)
	for _, ir := range r.wr {
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
	for _, ir := range r.wr {
		ir.CloseIdleConnections()
	}
}
