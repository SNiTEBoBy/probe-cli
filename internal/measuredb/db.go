package measuredb

// This file contains the definition of DB.

// DB is the database where we store measurements.
type DB interface {
	/*
	 * Straightforward table operations
	 */

	// InsertIntoConnection inserts an event into the connection table.
	InsertIntoConnection(c *ConnectionEvent)

	// SelectAllFromConnection returns all connection events.
	SelectAllFromConnection() []*ConnectionEvent

	// InsertIntoTLSHandshake inserts an event into the TLS handshake table.
	InsertIntoTLSHandshake(th *TLSHandshakeEvent)

	// SelectAllFromTLSHandshake returns all TLS handshakes.
	SelectAllFromTLSHandshake() []*TLSHandshakeEvent

	// InsertIntoLookupHost inserts an event into the LookupHost table.
	InsertIntoLookupHost(lh *LookupHostEvent)

	// SelectAllFromLookupHost returns all host lookups.
	SelectAllFromLookupHost() []*LookupHostEvent

	// InsertIntoHTTPRoundTrip inserts an event into the HTTP round trip table.
	InsertIntoHTTPRoundTrip(rt *HTTPRoundTripEvent)

	// SelectAllFromHTTPRoundTrip returns all HTTP round trips.
	SelectAllFromHTTPRoundTrip() []*HTTPRoundTripEvent

	// InsertIntoDomainEndpoint inserts an event into the domain-endpoint table.
	InsertIntoDomainEndpoint(v *DomainEndpointBinding)

	// SelectAllFromDomainEndpoint returns all domain-endpoint info.
	SelectAllFromDomainEndpoint() []*DomainEndpointBinding

	// InsertIntoDNSRoundTrip inserts an event into
	// the dns round trip table.
	InsertIntoDNSRoundTrip(v *DNSRoundTripEvent)

	// SelectAllFromDNSRoundTrip returns all dns round trip info.
	SelectAllFromDNSRoundTrip() []*DNSRoundTripEvent

	// InsertIntoHTTPRoundTripURL inserts an event into
	// the http-round-trip-url table.
	InsertIntoHTTPRoundTripURL(v *HTTPRoundTripURLBinding)

	// SelectAllFromHTTPRoundTripURL returns all the
	// entries of the http-round-trip-url table.
	SelectAllFromHTTPRoundTripURL() []*HTTPRoundTripURLBinding

	// InsertIntoTestHelperMeasurement inserts an event into
	// the test-helper-measurement table.
	InsertIntoTestHelperMeasurement(v *TestHelperMeasurement)

	// SelectAllFromTestHelperMeasurement returns all the
	// entries of the test-helper-measurement table.
	SelectAllFromTestHelperMeasurement() []*TestHelperMeasurement

	// InsertIntoLookupHTTPS inserts an event into the
	// lookup-https table for HTTPS queries.
	InsertIntoLookupHTTPS(v *LookupHTTPSEvent)

	// SelectAllFromLookupHTTPS returns the lookup-https table content.
	SelectAllFromLookupHTTPS() []*LookupHTTPSEvent

	/*
	 * Support for precise HTTP round trip measurements
	 *
	 * We say we support this kind of measurements when the DB is
	 * able to enforce the HTTPTransport to perform a single round
	 * trip at a time. When this happens, we can confidently get
	 * the HTTPRoundTripID when dialing and be sure that it refers
	 * to the current round trip. In turn, this guarantee means
	 * that we can join events by their round trip ID.
	 *
	 * The downside of this strategy is that we cannot have
	 * parallel HTTP round trips using the same DB.
	 *
	 * This strategy is optional because there is value in collecting
	 * measuredb stats even without this constraint. For example,
	 * say we use measuredb for our DoH resolvers. Then, we have data
	 * on the blocking of DoH resolver endpoints that we can submit
	 * to the OONI collector as a set of measurements.
	 */

	// SupportsPreciseHTTPRoundTripMeasurements returns true when the
	// database supports precise round trip measurements. If there
	// is no support for this functionality, several queries are
	// not possible and we can only collect endpoint stats.
	SupportsPreciseHTTPRoundTripMeasurements() bool

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

	/*
	 * Support for testing untested endpoints
	 *
	 * If we let the dialer-resolver-connector-transport
	 * perform measurements, we naturally miss a bunch of
	 * endpoints we could measure. Say we are measuring
	 * dns.google with endpoints 8.8.8.8:443 and 8.8.4.4:443
	 * on TCP and QUIC. Say that the first endpoint works
	 * for TCP. Then we are not testing the other endpoint
	 * using TCP and QUIC. So, what we do is that we record
	 * all the available endpoints during dial. Then, we
	 * have support for extracting untested endpoints along
	 * with the support for building the configuration for
	 * testing them (endpoints, domains, SNI, etc.).
	 *
	 * The most important feature we need database-wise to
	 * do that is support for extracting all the untested
	 * endpoints, so that we can measure them.
	 */

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
	RemoveUntestedEndpoints() ([]*DomainEndpointBinding, error)
}
