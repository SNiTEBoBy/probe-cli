package measuredb

// This code contains code to build views out of the database content.

import "errors"

// HTTPRoundTripView is an HTTP round trip centric view
// where every structure is an HTTP round trip.
type HTTPRoundTripView struct {
	URL             string
	EndpointID      int64
	HTTPRoundTripID int64
	DNSRoundTrip    []*DNSRoundTripEvent
	LookupHost      []*LookupHostEvent
	HTTPRoundTrip   *HTTPRoundTripEvent
	Endpoint        *HTTPEndpointView
}

var (
	errNoDatabaseSupport    = errors.New("no database support")
	errTooManyTLSHandshakes = errors.New("too many TLS handshakes")
)

// NewHTTPRoundTripView attempts to build an HTTPRoundTripView
// on top of a given database. This is only possible when the
// database supports precise HTTP round trip measurements. The
// return value is either the view or an error. This function
// may return an empty list even in case of success. This condition
// occurs when there are no relevant data inside the DB.
func NewHTTPRoundTripView(db DB) ([]*HTTPRoundTripView, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	var out []*HTTPRoundTripView
	for _, rtx := range db.SelectAllFromHTTPRoundTrip() {
		eview, err := NewHTTPEndpointView(db, rtx.HTTPRoundTripID, rtx.EndpointID)
		if err != nil {
			return nil, err
		}
		lh, _ := selectLookupHostWithRoundTripIP(db, rtx.HTTPRoundTripID)
		dnsrt, _ := selectDNSRoundTripWithRoundTripIP(db, rtx.HTTPRoundTripID)
		out = append(out, &HTTPRoundTripView{
			URL:             rtx.RequestURL.String(),
			EndpointID:      rtx.EndpointID,
			HTTPRoundTripID: rtx.HTTPRoundTripID,
			DNSRoundTrip:    dnsrt, // nil means none in this round trip
			LookupHost:      lh,    // same as above
			HTTPRoundTrip:   rtx,
			Endpoint:        eview,
		})
	}
	return out, nil
}

// HTTPEndpointView is a view of all the events occurring to
// an endpoint identified by a given endpoint ID.
type HTTPEndpointView struct {
	NetworkEvents []*ConnectionEvent
	TLSHandshake  *TLSHandshakeEvent
}

// NewHTTPEndpointView attempts to build an HTTPEndpointView
// on top of a given database. This is only possible when the
// database supports precise HTTP round trip measurements. The
// return value is either the view or an error. This function
// may return an empty list even in case of success. This condition
// occurs when there are no relevant data inside the DB.
//
// CAVEAT: This view excludes endpoint events occurring outside
// of a given round trip. So, persistent connections will appear
// out of the blue without connect or handshake. Likewise, we
// won't see DNS resolutions when we don't perform them in a round trip.
func NewHTTPEndpointView(db DB, roundTripID, endpointID int64) (*HTTPEndpointView, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	out := &HTTPEndpointView{}
	for _, conn := range db.SelectAllFromConnection() {
		if endpointID == conn.EndpointID && roundTripID == conn.HTTPRoundTripID {
			out.NetworkEvents = append(out.NetworkEvents, conn)
		}
	}
	for _, thx := range db.SelectAllFromTLSHandshake() {
		if endpointID == thx.EndpointID && roundTripID == thx.HTTPRoundTripID {
			if out.TLSHandshake != nil {
				// We expect to see a maximum of one TCP handshakes
				// during a round trip. If we see more than one this
				// is a bug in how we create/parse the data.
				return nil, errTooManyTLSHandshakes
			}
			out.TLSHandshake = thx
		}
	}
	return out, nil
}

// HTTPURLView is a view where we merge HTTPRoundTripView
// instances that use the same HTTP/HTTPS URL.
type HTTPURLView struct {
	URL        string
	LookupHost []*LookupHostEvent
	Endpoints  []*HTTPRoundTripView
}

// NewHTTPURLView builds a list of URLView instances
// from a list of HTTPRoundTripView instances.
func NewHTTPURLView(vv ...*HTTPRoundTripView) (out []*HTTPURLView) {
	m := make(map[string][]*HTTPRoundTripView)
	for _, v := range vv {
		m[v.URL] = append(m[v.URL], v)
	}
	for k, v := range m {
		out = append(out, &HTTPURLView{
			URL:        k,
			LookupHost: viewMergeLookupHost(v),
			Endpoints:  v,
		})
	}
	return out
}

func viewMergeLookupHost(vv []*HTTPRoundTripView) (out []*LookupHostEvent) {
	for _, rtx := range vv {
		if rtx.LookupHost != nil {
			out = append(out, rtx.LookupHost...)
		}
	}
	return
}
