package measuredb

// THIS FILE IS UNUSED

/*
// GroupByRoundTripIDEntry is the entry of a list that groups
// all measurements by their round trip ID.
type GroupByRoundTripIDEntry struct {
	ID             int64
	Connection     []*Connection
	DomainEndpoint []*DomainEndpoint
	TLSHandshake   []*TLSHandshake
	LookupHost     []*LookupHost
	HTTPRoundTrip  []*HTTPRoundTrip
}

// groupByRoundTripIDList allows implementing sorting
type groupByRoundTripIDList []*GroupByRoundTripIDEntry

func (vl groupByRoundTripIDList) Len() int {
	return len(vl)
}

func (vl groupByRoundTripIDList) Swap(i, j int) {
	vl[i], vl[j] = vl[j], vl[i]
}

// groupByRoundTripIDSortByRoundTripID allows implementing sorting
type groupByRoundTripIDSortByRoundTripID struct {
	groupByRoundTripIDList
}

func (v groupByRoundTripIDSortByRoundTripID) Less(i, j int) bool {
	return v.groupByRoundTripIDList[i].ID < v.groupByRoundTripIDList[j].ID
}

// GroupByRoundTripID groups measurements by HTTP round trip ID.
//
// This function filters out all the events with round trip ID negative
// or equal to zero. These values mean we are not tracking the round
// trip ID. All other values are grouped by round trip ID.
//
// CAVEAT: this functionality only works if the database supports
// precise grouping by HTTP round trip. Therefore, it there is
// no support, you will always get back an empty list.
//
// If not empty, the returned list is sorted by ascending ID.
func GroupByRoundTripID(db DB) []*GroupByRoundTripIDEntry {
	m := make(map[int64]*GroupByRoundTripIDEntry)

	for _, e := range db.SelectAllFromConnection() {
		id := e.RoundTripID
		if id < 1 {
			continue // no round trip ID
		}
		if m[id] == nil {
			m[id] = &GroupByRoundTripIDEntry{}
		}
		m[id].Connection = append(m[id].Connection, e)
	}

	for _, e := range db.SelectAllFromDomainEndpoint() {
		id := e.RoundTripID
		if id < 1 {
			continue // no round trip ID
		}
		if m[id] == nil {
			m[id] = &GroupByRoundTripIDEntry{}
		}
		m[id].DomainEndpoint = append(m[id].DomainEndpoint, e)
	}

	for _, e := range db.SelectAllFromTLSHandshake() {
		id := e.RoundTripID
		if id < 1 {
			continue // no round trip ID
		}
		if m[id] == nil {
			m[id] = &GroupByRoundTripIDEntry{}
		}
		m[id].TLSHandshake = append(m[id].TLSHandshake, e)
	}

	for _, e := range db.SelectAllFromLookupHost() {
		id := e.RoundTripID
		if id < 1 {
			continue // no round trip ID
		}
		if m[id] == nil {
			m[id] = &GroupByRoundTripIDEntry{}
		}
		m[id].LookupHost = append(m[id].LookupHost, e)
	}

	for _, e := range db.SelectAllFromHTTPRoundTrip() {
		id := e.RoundTripID
		if id < 1 {
			continue // no round trip ID
		}
		if m[id] == nil {
			m[id] = &GroupByRoundTripIDEntry{}
		}
		m[id].HTTPRoundTrip = append(m[id].HTTPRoundTrip, e)
	}

	for id, info := range m {
		info.ID = id // set ID for clarity
	}

	var out []*GroupByRoundTripIDEntry
	for _, e := range m {
		out = append(out, e)
	}
	sort.Sort(groupByRoundTripIDSortByRoundTripID{out})
	return out
}

// GroupHTTPRoundTripByEndpointID returns an unsorted
// list of HTTPRoundTrip grouped by endpoint ID.
func GroupHTTPRoundTripByEndpointID(v []*HTTPRoundTrip) (out [][]*HTTPRoundTrip) {
	m := make(map[int64][]*HTTPRoundTrip)
	for _, e := range v {
		m[e.RoundTripID] = append(m[e.RoundTripID], e)
	}
	for _, e := range m {
		out = append(out, e)
	}
	return
}

// GroupTLSHandshakeByEndpointID returns an unsorted
// list of TLSHandshake grouped by endpoint ID.
func GroupTLSHandshakeByEndpointID(v []*TLSHandshake) (out [][]*TLSHandshake) {
	m := make(map[int64][]*TLSHandshake)
	for _, e := range v {
		m[e.RoundTripID] = append(m[e.RoundTripID], e)
	}
	for _, e := range m {
		out = append(out, e)
	}
	return
}

// GroupConnectionByEndpointID returns an unsorted
// list of Connection grouped by endpoint ID.
func GroupConnectionByEndpointID(v []*Connection) (out [][]*Connection) {
	m := make(map[int64][]*Connection)
	for _, e := range v {
		m[e.RoundTripID] = append(m[e.RoundTripID], e)
	}
	for _, e := range m {
		out = append(out, e)
	}
	return
}
*/
