package measuredb

// This file contains "queries" functions to "select" specific
// subsets of the whole set of results in the DB.

import "errors"

var errHTTPRoundTripNotFound = errors.New("cannot find HTTPRoundTrip")

func selectHTTPRoundTripWithRoundTripID(db DB, id int64) (*HTTPRoundTripEvent, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	for _, rtx := range db.SelectAllFromHTTPRoundTrip() {
		if id == rtx.HTTPRoundTripID {
			return rtx, nil
		}
	}
	return nil, errHTTPRoundTripNotFound
}

var errTLSHandshakeNotFound = errors.New("cannot find TLSHandshake")

func selectTLSHandshakeWithRoundTripID(db DB, id int64) (*TLSHandshakeEvent, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	for _, thx := range db.SelectAllFromTLSHandshake() {
		if id == thx.HTTPRoundTripID {
			return thx, nil
		}
	}
	return nil, errTLSHandshakeNotFound
}

func selectLookupHostWithRoundTripIP(db DB, id int64) ([]*LookupHostEvent, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	var out []*LookupHostEvent
	for _, lhx := range db.SelectAllFromLookupHost() {
		if id == lhx.HTTPRoundTripID {
			out = append(out, lhx)
		}
	}
	return out, nil
}

func selectDNSRoundTripWithRoundTripIP(db DB, id int64) ([]*DNSRoundTripEvent, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	var out []*DNSRoundTripEvent
	for _, drt := range db.SelectAllFromDNSRoundTrip() {
		if id == drt.HTTPRoundTripID {
			out = append(out, drt)
		}
	}
	return out, nil
}

var errHTTPRoundTripURLNotFound = errors.New("cannot find HTTPRoundTripURL")

func selectURLFromHTTPRoundTripURLWithCurrentRoundTrip(db DB) (string, error) {
	if !db.SupportsPreciseHTTPRoundTripMeasurements() {
		return "", errNoDatabaseSupport
	}
	id := db.HTTPRoundTripID()
	for _, rtx := range db.SelectAllFromHTTPRoundTripURL() {
		if id == rtx.HTTPRoundTripID {
			return rtx.URL.String(), nil
		}
	}
	return "", errHTTPRoundTripURLNotFound
}
