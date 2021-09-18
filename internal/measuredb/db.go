package measuredb

// DB is the database where we store measurements.
type DB interface {
	// InsertIntoConnection inserts an event into the connection table.
	InsertIntoConnection(c *Connection)

	// SelectAllFromConnection returns all connection events.
	SelectAllFromConnection() []*Connection

	// InsertIntoTLSHandshake inserts an event into the TLS handshake table.
	InsertIntoTLSHandshake(th *TLSHandshake)

	// SelectAllFromTLSHandshake returns all TLS handshakes.
	SelectAllFromTLSHandshake() []*TLSHandshake

	// InsertIntoLookupHost inserts an event into the LookupHost table.
	InsertIntoLookupHost(lh *LookupHost)

	// SelectAllFromLookupHost returns all host lookups.
	SelectAllFromLookupHost() []*LookupHost

	// InsertIntoHTTPRoundTrip inserts an event into the HTTP round trip table.
	InsertIntoHTTPRoundTrip(rt *HTTPRoundTrip)

	// SelectAllFromHTTPRoundTrip returns all HTTP round trips.
	SelectAllFromHTTPRoundTrip() []*HTTPRoundTrip

	// InsertIntoDomainEndpoint inserts an event into the domain-endpoint table.
	InsertIntoDomainEndpoint(v *DomainEndpoint)

	// SelectAllFromDomainEndpoint returns all domain-endpoint info.
	SelectAllFromDomainEndpoint() []*DomainEndpoint

	// SupportsPreciseRoundTripMeasurements returns true when the
	// database supports precise round trip measurements. If there
	// is no support for this functionality, several queries are
	// not possible and we can only collect endpoint stats.
	SupportsPreciseRoundTripMeasurements() bool

	// OnEnterHTTPRoundTrip is called by the HTTPTransport when
	// we begin a new HTTP round trip. If you wish to separate the
	// measurements of each round trip (aka "precise round trip
	// measurements"), this is your chance to stop any concurrent
	// attempt at round tripping.
	OnEnterHTTPRoundTrip()

	// OnLeaveHTTPRoundTrip is called by the HTTPTransport when
	// we end an HTTP round trip. If you wish to separate the
	// measurements of each round trip, here is where you should
	// allow the next concurrent attempt to resume.
	OnLeaveHTTPRoundTrip()

	// HTTPRoundTripID returns a positive number indicating
	// the current round trip ID, when we are measuring
	// precise HTTP round trips (see OnEnterHTTPRoundTrip and
	// OnLeaveHTTPRoundTrip). A zero or negative return
	// value means we are not able to precisely measure the
	// round trips. (Yes, we should not overflow an int64.)
	HTTPRoundTripID() int64

	// EndpointID is like HTTPRoundTripDB except that it
	// uniquely identify a remote endpoint. The same caveat
	// regarding precise HTTP round trips also apply.
	EndpointID() int64

	// OnTryEndpoint informs the database that we are
	// about to try and use a new endpoint. This function
	// is called when dialing for QUIC and TCP. This
	// function only works reliably with precise HTTP
	// round trip measurements. It effectively increases
	// the current endpoint ID by one.
	OnTryEndpoint(network, address string)

	// RemoveUntestedEndpoints is a destructive operation
	// that removes from the database all the TCP/QUIC
	// endpoints that have not been tested. We say that
	// an endpoint has not been tested when we never
	// attempt for it a TCP connect or a QUIC handshake.
	//
	// This function returns an error if the database
	// does not support precise HTTP round trip
	// measurements. If there are no errors, this
	// function returns the list of removed endpoints,
	// which may be empty if there are none.
	RemoveUntestedEndpoints() ([]*DomainEndpoint, error)

	// InsertIntoDNSRoundTrip inserts an event into
	// the dns round trip table.
	InsertIntoDNSRoundTrip(v *DNSRoundTrip)

	// SelectAllFromDNSRoundTrip returns all dns round trip info.
	SelectAllFromDNSRoundTrip() []*DNSRoundTrip
}
