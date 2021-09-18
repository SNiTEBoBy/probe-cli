package measuredb

// This file contains code for creating a Dialer that has
// measuredb measurement capabilities.

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
// 2. query the test helper to augment our view of the endpoint to test;
//
// 3. build a list of TCP/QUIC endpoints to test;
//
// 4. insert such a list into the DomainEndpoint table;
//
// 5. attempt to TCP connect all of the TCP endpoints and return
// at the first success. All untested endpoints are still accessible
// later via the DomainEndpoint table, which gives us a chance to
// measure anyone of them at a later time. We skip all QUIC endpoints
// by default: they are always processed as untested endpoints.
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

// The constants indicate whether we discovered an endpoint
// in the probe or thanks to the test helper.
var (
	EndpointOriginProbe      = "probe"
	EndpointOriginTestHelper = "th"
)

// DomainEndpointBinding maps a domain to one of its endpoints.
//
// This data structure contains enough information to re-test
// an untested endpoint at a later time.
//
// CAVEAT: HTTPRoundTripID is only meaningful when the
// underlying DB supports precise round trip measurements.
type DomainEndpointBinding struct {
	// HTTPRoundTripID is the HTTP round trip ID
	HTTPRoundTripID int64

	// Origin indicates the endpoint origin ("th" or "probe")
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

func domainEndpointsAsEndpoints(des []*DomainEndpointBinding) (out []string) {
	for _, de := range des {
		switch de.Network {
		case "tcp":
		default:
			// Do not pass to the test helper QUIC addresses. When we
			// will upgrade the test helper we can change this.
			continue
		}
		out = append(out, de.Address)
	}
	return
}

func newDomainEndpoints(db DB,
	origin, domain, network, port string, addrs ...string) (out []*DomainEndpointBinding) {
	for _, addr := range addrs {
		out = append(out, &DomainEndpointBinding{
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
	domain, network, port string, endpoints []*DomainEndpointBinding,
	resp *TestHelperMeasurement) []*DomainEndpointBinding {
	m := make(map[string]bool)
	for _, epnt := range endpoints {
		m[epnt.Address] = true
	}
	for _, entry := range resp.DNSAddrs {
		if address := net.JoinHostPort(entry, port); !m[address] {
			endpoints = append(endpoints, &DomainEndpointBinding{
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

func httpsReplyContainsH3(https netxlite.HTTPS) bool {
	for _, alpn := range https.ALPN() {
		switch alpn {
		case "h3":
			return true
		}
	}
	return false
}

func httpsReplyGetAllIPAddrs(https netxlite.HTTPS) (addrs []string) {
	addrs = append(addrs, https.IPv4Hint()...)
	addrs = append(addrs, https.IPv6Hint()...)
	return
}

func domainEndpointsMergeHTTPS(db DB, origin, domain, port string,
	endpoints []*DomainEndpointBinding, https netxlite.HTTPS) []*DomainEndpointBinding {
	if !httpsReplyContainsH3(https) {
		return endpoints
	}
	for _, addr := range httpsReplyGetAllIPAddrs(https) {
		endpoints = append(endpoints, &DomainEndpointBinding{
			HTTPRoundTripID: db.HTTPRoundTripID(),
			Origin:          origin,
			Domain:          domain,
			Network:         "h3",
			Address:         net.JoinHostPort(addr, port),
			EndpointID:      0,   // for now
			conn:            nil, // for now
		})
	}
	return endpoints
}

func (d *dialerDB) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	domain, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	// 1. issue normal lookup A+AAAA query
	addrs, err := d.resolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}
	endpoints := newDomainEndpoints(
		d.db, EndpointOriginProbe, domain, network, port, addrs...)

	// 2. issue HTTPS/SVCB query if possible
	https, err := d.resolver.LookupHTTPSWithoutRetry(ctx, domain)
	if err == nil {
		endpoints = domainEndpointsMergeHTTPS(
			d.db, EndpointOriginProbe, domain, port, endpoints, https)
	}

	// 3. query the test helper if possible
	thResp, err := d.th.Run(ctx, domainEndpointsAsEndpoints(endpoints))
	if err == nil {
		endpoints = domainEndpointsMergeTestHelperEndpoints(
			d.db, domain, network, port, endpoints, thResp)
	}

	return d.dialLoop(ctx, endpoints)
}

func (d *dialerDB) dialLoop(
	ctx context.Context, endpoints []*DomainEndpointBinding) (net.Conn, error) {
	// TODO(bassosimone): could we run these steps in parallel
	// without screwing up with connection ID assignation?
	for _, epnt := range endpoints {
		switch epnt.Network {
		case "tcp", "udp":
		default:
			// Skip QUIC endpoints. They will always be tested
			// as untested endpoints after we've tried TCP.
			continue
		}
		conn, err := d.connector.DialContext(ctx, epnt.Network, epnt.Address)
		// Implementation note: we MUST get the endpoint ID HERE rather
		// than before DialContext because we increment the endpoint
		// ID inside of the connector.DialContext function.
		//
		// TODO(bassosimone): it would be cool if the returned connection
		// contained its own endpoint ID so we don't need this here
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
