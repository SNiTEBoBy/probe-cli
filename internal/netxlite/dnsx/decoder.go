package dnsx

import (
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// HTTPS is an HTTPS reply.
type HTTPS = model.HTTPS

type https struct {
	alpn []string
}

var _ HTTPS = &https{}

func (h *https) ALPN() []string {
	return h.alpn
}

// The Decoder decodes a DNS replies.
type Decoder interface {
	// DecodeLookupHost decodes an A or AAAA reply.
	DecodeLookupHost(qtype uint16, data []byte) ([]string, error)

	// DecodeHTTPS decodes an HTTPS reply.
	DecodeHTTPS(data []byte) (HTTPS, error)
}

// MiekgDecoder uses github.com/miekg/dns to implement the Decoder.
type MiekgDecoder struct{}

func (d *MiekgDecoder) parseReply(data []byte) (*dns.Msg, error) {
	reply := new(dns.Msg)
	if err := reply.Unpack(data); err != nil {
		return nil, err
	}
	// TODO(bassosimone): map more errors to net.DNSError names
	// TODO(bassosimone): add support for lame referral.
	switch reply.Rcode {
	case dns.RcodeSuccess:
		return reply, nil
	case dns.RcodeNameError:
		return nil, errorsx.ErrOODNSNoSuchHost
	case dns.RcodeRefused:
		return nil, errorsx.ErrOODNSRefused
	default:
		return nil, errorsx.ErrOODNSMisbehaving
	}
}

func (d *MiekgDecoder) DecodeHTTPS(data []byte) (HTTPS, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
	}
	out := &https{}
	for _, answer := range reply.Answer {
		switch avalue := answer.(type) {
		case *dns.HTTPS:
			for _, v := range avalue.Value {
				switch extv := v.(type) {
				case *dns.SVCBAlpn:
					out.alpn = append(out.alpn, extv.Alpn...)
				}
			}
		}
	}
	if len(out.alpn) <= 0 {
		return nil, errorsx.ErrOODNSNoAnswer
	}
	return out, nil
}

func (d *MiekgDecoder) DecodeLookupHost(qtype uint16, data []byte) ([]string, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, answer := range reply.Answer {
		switch qtype {
		case dns.TypeA:
			if rra, ok := answer.(*dns.A); ok {
				ip := rra.A
				addrs = append(addrs, ip.String())
			}
		case dns.TypeAAAA:
			if rra, ok := answer.(*dns.AAAA); ok {
				ip := rra.AAAA
				addrs = append(addrs, ip.String())
			}
		}
	}
	if len(addrs) <= 0 {
		return nil, errorsx.ErrOODNSNoAnswer
	}
	return addrs, nil
}

var _ Decoder = &MiekgDecoder{}
