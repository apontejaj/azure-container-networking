package nmagent

import (
	"encoding/xml"
	"fmt"
	"net"
)

type IPAddress net.IP

func (h *IPAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var ipStr string
	if err := d.DecodeElement(&ipStr, &start); err != nil {
		return err
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}

	*h = IPAddress(ip)
	return nil
}

func (h *IPAddress) UnmarshalXMLAttr(attr xml.Attr) error {
	ipStr := attr.Value
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}

	*h = IPAddress(ip)
	return nil
}

func (h IPAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(net.IP(h).String(), start)
}

func (h IPAddress) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{
		Name:  name,
		Value: net.IP(h).String(),
	}, nil
}
