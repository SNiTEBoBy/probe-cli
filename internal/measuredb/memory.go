package measuredb

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
	connection     []*Connection
	domainEndpoint []*DomainEndpoint
	tlsHandshake   []*TLSHandshake
	lookupHost     []*LookupHost
	httpRoundTrip  []*HTTPRoundTrip

	// mu provides mutual exclusion when accessing data
	mu sync.Mutex

	// rtBarrier ensures each round trip runs separately
	rtBarrier sync.Mutex

	// rtID is the next round trip ID
	rtID *atomicx.Int64

	// eID is the next remote endpoint ID
	eID *atomicx.Int64
}

func (db *memoryDB) InsertIntoConnection(c *Connection) {
	db.mu.Lock()
	db.connection = append(db.connection, c)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromConnection() (out []*Connection) {
	db.mu.Lock()
	out = append(out, db.connection...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoTLSHandshake(th *TLSHandshake) {
	db.mu.Lock()
	db.tlsHandshake = append(db.tlsHandshake, th)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromTLSHandshake() (out []*TLSHandshake) {
	db.mu.Lock()
	out = append(out, db.tlsHandshake...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoLookupHost(lh *LookupHost) {
	db.mu.Lock()
	db.lookupHost = append(db.lookupHost, lh)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromLookupHost() (out []*LookupHost) {
	db.mu.Lock()
	out = append(out, db.lookupHost...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoHTTPRoundTrip(rt *HTTPRoundTrip) {
	db.mu.Lock()
	db.httpRoundTrip = append(db.httpRoundTrip, rt)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromHTTPRoundTrip() (out []*HTTPRoundTrip) {
	db.mu.Lock()
	out = append(out, db.httpRoundTrip...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) InsertIntoDomainEndpoint(v *DomainEndpoint) {
	db.mu.Lock()
	db.domainEndpoint = append(db.domainEndpoint, v)
	db.mu.Unlock()
}

func (db *memoryDB) SelectAllFromDomainEndpoint() (out []*DomainEndpoint) {
	db.mu.Lock()
	out = append(out, db.domainEndpoint...)
	db.mu.Unlock()
	return
}

func (db *memoryDB) SupportsPreciseRoundTripMeasurements() bool {
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

func (db *memoryDB) RemoveUntestedEndpoints() ([]*DomainEndpoint, error) {
	defer db.mu.Unlock()
	db.mu.Lock()
	var (
		removed []*DomainEndpoint
		left    []*DomainEndpoint
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
