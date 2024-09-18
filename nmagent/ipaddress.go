package nmagent

import (
	"encoding/xml"
	"net"

	"github.com/pkg/errors"
)

type IPAddress net.IP

func (h *IPAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var ipStr string
	if err := d.DecodeElement(&ipStr, &start); err != nil {
		return errors.Wrap(err, "Decoding IP address")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &net.ParseError{Type: "IP address", Text: ipStr}
	}

	*h = IPAddress(ip)
	return nil
}

func (h *IPAddress) UnmarshalXMLAttr(attr xml.Attr) error {
	ipStr := attr.Value
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &net.ParseError{Type: "IP address", Text: ipStr}
	}

	*h = IPAddress(ip)
	return nil
}

func (h IPAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	err := e.EncodeElement(net.IP(h).String(), start)
	return errors.Wrap(err, "Encoding IP address")
}

func (h IPAddress) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{
		Name:  name,
		Value: net.IP(h).String(),
	}, nil
}
