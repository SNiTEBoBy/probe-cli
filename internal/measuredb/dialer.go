package measuredb

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewDialer creates a new Dialer with measuredb capabilities.
//
// Arguments
//
// - logger is the logger to use;
//
// - db is the database to use;
//
// - th is the test helper to use (pass &NullTestHelper{} if you
// don't want to use any test helper at all);
//
// - resolver is the resolver to use;
//
// - connector is the TCP/UDP connector to use.
//
// All arguments are mandatory.
//
// The return value is a netxlite.Dialer wrapper that adds to the
// original dialer measuredb and logging capabilities.
//
// DialContext algorithm
//
// 1. perform DNS queries (A, AAAA, possibly SVCB);
//
// 2. call the test helper to augment our view of the endpoint to test;
//
// 3. build a list of TCP/QUIC endpoints to test;
//
// 4. insert such a list into the DomainEndpoint table;
//
// 5. attempt to TCP connect all of the TCP endpoints and return
// at the first success. All untested endpoints are still accessible
// later via the DomainEndpoint table, which gives us a chance to
// measure anyone of them at a later time.
func NewDialer(logger netxlite.Logger, db DB, th TestHelper,
	resolver netxlite.Resolver, connector netxlite.Connector) netxlite.Dialer {
	return &netxlite.DialerLogger{
		Dialer: &dialerDB{
			connector: connector,
			db:        db,
			logger:    logger,
			resolver:  resolver,
			th:        th,
		},
		Logger: logger,
	}
}

type dialerDB struct {
	connector netxlite.Connector
	db        DB
	logger    netxlite.Logger
	resolver  netxlite.Resolver
	th        TestHelper
}

func (d *dialerDB) CloseIdleConnections() {
	d.resolver.CloseIdleConnections()
}

// ErrDial indicates that a dial operation failed. Because we
// are measuring via tracing, it does not matter to report what
// error actually occurred to the caller (for now at least).
var ErrDial = errors.New("dial failed")

var (
	EndpointOriginProbe      = "probe"
	EndpointOriginTestHelper = "th"
)

// DomainEndpoint maps a domain to one of its endpoints.
//
// This data structure contains enough information to test
// the endpoint at hand at a later time.
//
// CAVEAT: HTTPRoundTripID is only meaningful when the
// underlying DB supports precise round trip measurements.
type DomainEndpoint struct {
	// HTTPRoundTripID is the HTTP round trip ID
	HTTPRoundTripID int64

	// Origin indicates the endpoint origin
	Origin string

	// Domain
	Domain string

	// Endpoint
	Network    string
	Address    string
	EndpointID int64

	// temporary storage for conn (see below)
	conn net.Conn `json:"-"`
}

func domainEndpointsAsEndpoints(des []*DomainEndpoint) (out []string) {
	for _, de := range des {
		out = append(out, de.Address)
	}
	return
}

func newDomainEndpoints(db DB,
	origin, domain, network, port string, addrs ...string) (out []*DomainEndpoint) {
	for _, addr := range addrs {
		out = append(out, &DomainEndpoint{
			HTTPRoundTripID: db.HTTPRoundTripID(),
			Origin:          origin,
			Domain:          domain,
			Network:         network,
			Address:         net.JoinHostPort(addr, port),
			EndpointID:      0,   // for now
			conn:            nil, // for now
		})
	}
	return
}

func domainEndpointsMergeTestHelperEndpoints(db DB,
	domain, network, port string, endpoints []*DomainEndpoint,
	resp *TestHelperMeasurement) []*DomainEndpoint {
	m := make(map[string]bool)
	for _, epnt := range endpoints {
		m[epnt.Address] = true
	}
	for _, entry := range resp.DNSAddrs {
		address := net.JoinHostPort(entry, port)
		if !m[entry] {
			endpoints = append(endpoints, &DomainEndpoint{
				HTTPRoundTripID: db.HTTPRoundTripID(),
				Origin:          EndpointOriginTestHelper,
				Domain:          domain,
				Network:         network,
				Address:         address,
				EndpointID:      0,   // filled later
				conn:            nil, // same
			})
		}
	}
	return endpoints
}

func (d *dialerDB) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	domain, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): we should run a SVCB query here
	addrs, err := d.resolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}
	endpoints := newDomainEndpoints(
		d.db, EndpointOriginProbe, domain, network, port, addrs...)
	thResp, err := d.th.Run(ctx, domainEndpointsAsEndpoints(endpoints))
	if err == nil {
		endpoints = domainEndpointsMergeTestHelperEndpoints(
			d.db, domain, network, port, endpoints, thResp)
	}
	return d.dialLoop(ctx, endpoints)
}

func (d *dialerDB) dialLoop(
	ctx context.Context, endpoints []*DomainEndpoint) (net.Conn, error) {
	// TODO(bassosimone): could we run these steps in parallel
	// without screwing up with connection ID assignation?
	for _, epnt := range endpoints {
		conn, err := d.connector.DialContext(ctx, epnt.Network, epnt.Address)
		// Implementation note: we MUST get the endpoint ID HERE rather
		// than before DialContext because we increment the endpoint
		// ID inside of the connector.DialContext function.
		epnt.EndpointID = d.db.EndpointID()
		if err == nil {
			epnt.conn = conn
			break
		}
	}
	var (
		conn     net.Conn
		finalErr = ErrDial
	)
	for _, epnt := range endpoints {
		if epnt.conn != nil {
			conn = epnt.conn
			finalErr = nil
		}
		d.db.InsertIntoDomainEndpoint(epnt)
	}
	return conn, finalErr
}
