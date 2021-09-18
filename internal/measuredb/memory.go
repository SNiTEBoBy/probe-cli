package measuredb

// This file contains the implementation of an in-memory DB

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

// NewMemoryDB creates a new DB that keeps content in memory.
//
// This database implements precise HTTP round trip measurements
// so it's suitable for building OONI measurements because it
// allows to uniquely attribute all events to HTTP round trips.
func NewMemoryDB() DB {
	return &memoryDB{
		rtID: &atomicx.Int64{},
		eID:  &atomicx.Int64{},
	}
}

type memoryDB struct {
	// These variables contain the DB "tables"
	connection       []*ConnectionEvent
	domainEndpoint   []*DomainEndpointBinding
	tlsHandshake     []*TLSHandshakeEvent
	lookupHost       []*LookupHostEvent
	httpRoundTrip    []*HTTPRoundTripEvent
	dnsRoundTrip     []*DNSRoundTripEvent
	httpRoundTripURL []*HTTPRoundTripURLBinding
	testHelperMeas   []*TestHelperMeasurement
	lookupHTTPS      []*LookupHTTPSEvent

	// mu provides mutual exclusion when accessing data
	mu sync.Mutex

	// rtBarrier ensures each round trip runs separately thus
	// allowing for precise HTTP round trip measurements
	rtBarrier sync.Mutex

	// rtID is the round trip ID
	rtID *atomicx.Int64

	// eID is the endpoint ID
	eID *atomicx.Int64
}

func (db *memoryDB) InsertIntoConnection(c *ConnectionEvent) {
	db.mu.Lock()
	db.connection = append(db.connection, c)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromConnection() (out []*ConnectionEvent) {
	db.mu.Lock()
	out = append(out, db.connection...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoTLSHandshake(th *TLSHandshakeEvent) {
	db.mu.Lock()
	db.tlsHandshake = append(db.tlsHandshake, th)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromTLSHandshake() (out []*TLSHandshakeEvent) {
	db.mu.Lock()
	out = append(out, db.tlsHandshake...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoLookupHost(lh *LookupHostEvent) {
	db.mu.Lock()
	db.lookupHost = append(db.lookupHost, lh)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromLookupHost() (out []*LookupHostEvent) {
	db.mu.Lock()
	out = append(out, db.lookupHost...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoHTTPRoundTrip(rt *HTTPRoundTripEvent) {
	db.mu.Lock()
	db.httpRoundTrip = append(db.httpRoundTrip, rt)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromHTTPRoundTrip() (out []*HTTPRoundTripEvent) {
	db.mu.Lock()
	out = append(out, db.httpRoundTrip...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoDomainEndpoint(v *DomainEndpointBinding) {
	db.mu.Lock()
	db.domainEndpoint = append(db.domainEndpoint, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromDomainEndpoint() (out []*DomainEndpointBinding) {
	db.mu.Lock()
	out = append(out, db.domainEndpoint...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) SupportsPreciseHTTPRoundTripMeasurements() bool {
	return true
}

func (db *memoryDB) OnEnterHTTPRoundTrip() {
	db.rtBarrier.Lock()
	db.rtID.Add(1) // MUST be _after_ the barrier
}

func (db *memoryDB) OnLeaveHTTPRoundTrip() {
	db.rtBarrier.Unlock()
}

func (db *memoryDB) HTTPRoundTripID() int64 {
	return db.rtID.Load()
}

func (db *memoryDB) EndpointID() int64 {
	return db.eID.Load()
}

func (db *memoryDB) OnTryEndpoint(network, address string) {
	db.eID.Add(1)
}

func (db *memoryDB) RemoveUntestedEndpoints() ([]*DomainEndpointBinding, error) {
	defer db.mu.Unlock()
	db.mu.Lock()
	var (
		removed []*DomainEndpointBinding
		left    []*DomainEndpointBinding
	)
	for _, dep := range db.domainEndpoint {
		// Reminder that an EndpointID <= 0 means "not supported"
		if dep.EndpointID > 0 {
			left = append(left, dep)
			continue
		}
		removed = append(removed, dep)
	}
	db.domainEndpoint = left
	return removed, nil
}

func (db *memoryDB) InsertIntoDNSRoundTrip(v *DNSRoundTripEvent) {
	db.mu.Lock()
	db.dnsRoundTrip = append(db.dnsRoundTrip, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromDNSRoundTrip() (out []*DNSRoundTripEvent) {
	db.mu.Lock()
	out = append(out, db.dnsRoundTrip...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoHTTPRoundTripURL(v *HTTPRoundTripURLBinding) {
	db.mu.Lock()
	db.httpRoundTripURL = append(db.httpRoundTripURL, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromHTTPRoundTripURL() (out []*HTTPRoundTripURLBinding) {
	db.mu.Lock()
	out = append(out, db.httpRoundTripURL...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoTestHelperMeasurement(v *TestHelperMeasurement) {
	db.mu.Lock()
	db.testHelperMeas = append(db.testHelperMeas, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromTestHelperMeasurement() (out []*TestHelperMeasurement) {
	db.mu.Lock()
	out = append(out, db.testHelperMeas...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoLookupHTTPS(v *LookupHTTPSEvent) {
	db.mu.Lock()
	db.lookupHTTPS = append(db.lookupHTTPS, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromLookupHTTPS() (out []*LookupHTTPSEvent) {
	db.mu.Lock()
	out = append(out, db.lookupHTTPS...)
	db.mu.Unlock()
	return
}
