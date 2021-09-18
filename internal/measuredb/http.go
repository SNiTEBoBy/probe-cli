package measuredb

// This file contains code to wrap HTTPTransports and
// get measuredb measurement capabilities

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

// WrapHTTPTransport wraps an HTTPTransport to add measuredb capabilities.
//
// RoundTrip algorithm
//
// 1. try to enforce precise HTTP round trip measurements (which
// depends on whether the database supports that);
//
// 2. register the round trip ID to URL binding in the DB;
//
// 3. do the actual round trip;
//
// 4. prepare the HTTP round trip event;
//
// 5. if no error, read a snapshot of the body;
//
// 6. save the round trip event and return.
func WrapHTTPTransport(db DB, txp netxlite.HTTPTransport) netxlite.HTTPTransport {
	return &httpTransportDB{HTTPTransport: txp, db: db}
}

type httpTransportDB struct {
	netxlite.HTTPTransport
	db DB
}

// HTTPRoundTripEvent contains information about an HTTP round trip.
//
// Note that EndpointID and RoundTripID only make sense when
// the DB we're using enforces precise HTTP round trips.
type HTTPRoundTripEvent struct {
	EndpointID           int64       // endpoint ID
	HTTPRoundTripID      int64       // HTTP round trip ID
	RequestMethod        string      // request method
	RequestURL           *url.URL    // request URL
	RequestHeader        http.Header // request headers
	Started              time.Time   // when we started
	Finished             time.Time   // when we finished
	Error                error       // error or nil
	ResponseStatus       int         // response status
	ResponseHeader       http.Header // response headers
	ResponseBodySnapshot []byte      // response body snapshot
}

// HTTPRoundTripURLBinding maps a round trip to its URL.
type HTTPRoundTripURLBinding struct {
	HTTPRoundTripID int64
	URL             *url.URL
}

// We only read a small snapshot of the body to keep measurements
// lean, since we're mostly interested in TLS interference nowadays
// but we'll also allow for reading more bytes from the conn.
const maxBodySnapshot = 1 << 11

func (txp *httpTransportDB) RoundTrip(req *http.Request) (*http.Response, error) {
	defer txp.db.OnLeaveHTTPRoundTrip()
	txp.db.OnEnterHTTPRoundTrip() // allow for precise round trip counting
	started := time.Now()
	txp.db.InsertIntoHTTPRoundTripURL(&HTTPRoundTripURLBinding{
		HTTPRoundTripID: txp.db.HTTPRoundTripID(),
		URL:             req.URL,
	})
	resp, err := txp.HTTPTransport.RoundTrip(req)
	rt := &HTTPRoundTripEvent{
		EndpointID:      txp.db.EndpointID(), // MUST be _after_ RoundTrip
		HTTPRoundTripID: txp.db.HTTPRoundTripID(),
		RequestMethod:   req.Method,
		RequestURL:      req.URL,
		RequestHeader:   req.Header,
		Started:         started,
	}
	if err != nil {
		rt.Finished = time.Now()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	rt.ResponseStatus = resp.StatusCode
	rt.ResponseHeader = resp.Header
	r := io.LimitReader(resp.Body, maxBodySnapshot)
	body, err := iox.ReadAllContext(req.Context(), r)
	if err != nil {
		// TODO(bassosimone): ensure we support unexpected EOF
		rt.Finished = time.Now()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	resp.Body = &httpTransportBody{ // allow for reading more if needed
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}
	rt.ResponseBodySnapshot = body
	rt.Finished = time.Now()
	txp.db.InsertIntoHTTPRoundTrip(rt)
	return resp, nil
}

type httpTransportBody struct {
	io.Reader
	io.Closer
}
