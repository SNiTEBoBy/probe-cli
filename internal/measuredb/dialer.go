package measuredb

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewDialer creates a new Dialer with measuredb capabilities.
func NewDialer(db DB, logger netxlite.Logger,
	resolver netxlite.Resolver, connector netxlite.Connector) netxlite.Dialer {
	return &netxlite.DialerLogger{
		Dialer: &dialerDB{
			DB:        db,
			logger:    logger,
			resolver:  resolver,
			connector: connector,
		},
		Logger: logger,
	}
}

type dialerDB struct {
	DB
	logger    netxlite.Logger
	resolver  netxlite.Resolver
	connector netxlite.Connector
}

func (d *dialerDB) CloseIdleConnections() {
	d.resolver.CloseIdleConnections()
}

// ErrDial indicates that a dial operation failed. Because we
// are measuring via tracing, it does not matter to report what
// error actually occurred to the caller (for now at least).
var ErrDial = errors.New("dial failed")

// DomainEndpoint maps a domain to one of its endpoints.
type DomainEndpoint struct {
	// RoundTripID is the HTTP round trip ID
	RoundTripID int64

	// Domain
	Domain string

	// Endpoint
	Network    string
	Address    string
	EndpointID int64

	// temporary storage for conn (see below)
	conn net.Conn `json:"-"`
}

func (d *dialerDB) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	domain, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := d.resolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}
	var endpoints []*DomainEndpoint
	for _, addr := range addrs {
		ap := net.JoinHostPort(addr, port)
		endpoints = append(endpoints, &DomainEndpoint{
			RoundTripID: d.DB.HTTPRoundTripID(),
			Domain:      domain,
			Network:     network,
			Address:     ap,
			EndpointID:  0,   // for now
			conn:        nil, // for now
		})
	}
	for _, epnt := range endpoints {
		conn, err := d.connector.DialContext(ctx, epnt.Network, epnt.Address)
		// Implementation note: we MUST get the endpoint ID HERE rather
		// than before DialContext because we increment the endpoint
		// ID inside of the connector.DialContext function.
		epnt.EndpointID = d.DB.EndpointID()
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
		d.DB.InsertIntoDomainEndpoint(epnt)
	}
	return conn, finalErr
}
