package measuredb

import "errors"

var errHTTPRoundTripNotFound = errors.New("cannot find HTTPRoundTrip")

func selectHTTPRoundTripWithRoundTripID(db DB, id int64) (*HTTPRoundTrip, error) {
	if !db.SupportsPreciseRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	for _, rtx := range db.SelectAllFromHTTPRoundTrip() {
		if id == rtx.RoundTripID {
			return rtx, nil
		}
	}
	return nil, errHTTPRoundTripNotFound
}

var errTLSHandshakeNotFound = errors.New("cannot find TLSHandshake")

func selectTLSHandshakeWithRoundTripID(db DB, id int64) (*TLSHandshake, error) {
	if !db.SupportsPreciseRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	for _, thx := range db.SelectAllFromTLSHandshake() {
		if id == thx.RoundTripID {
			return thx, nil
		}
	}
	return nil, errTLSHandshakeNotFound
}

func selectLookupHostWithRoundTripIP(db DB, id int64) ([]*LookupHost, error) {
	if !db.SupportsPreciseRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	var out []*LookupHost
	for _, lhx := range db.SelectAllFromLookupHost() {
		if id == lhx.RoundTripID {
			out = append(out, lhx)
		}
	}
	return out, nil
}

func selectDNSRoundTripWithRoundTripIP(db DB, id int64) ([]*DNSRoundTrip, error) {
	if !db.SupportsPreciseRoundTripMeasurements() {
		return nil, errNoDatabaseSupport
	}
	var out []*DNSRoundTrip
	for _, drt := range db.SelectAllFromDNSRoundTrip() {
		if id == drt.RoundTripID {
			out = append(out, drt)
		}
	}
	return out, nil
}
