package nmagent

import (
	"encoding/hex"
	"encoding/xml"
	"net"

	"github.com/pkg/errors"
)

const (
	MacAddressSize = 6
)

type MacAddress net.HardwareAddr

func (h *MacAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var macStr string
	if err := d.DecodeElement(&macStr, &start); err != nil {
		return errors.Wrap(err, "Decoding MAC address")
	}

	// Convert the string (without colons) into a valid MacAddress
	mac, err := hex.DecodeString(macStr)
	if err != nil {
		return &net.ParseError{Type: "MAC address", Text: macStr}
	}

	*h = MacAddress(mac)
	return nil
}

func (h *MacAddress) UnmarshalXMLAttr(attr xml.Attr) error {
	macStr := attr.Value
	mac, err := hex.DecodeString(macStr)
	if err != nil {
		return &net.ParseError{Type: "MAC address", Text: macStr}
	}

	*h = MacAddress(mac)
	return nil
}

func (h MacAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(h) != MacAddressSize {
		return &net.AddrError{Err: "invalid MAC address", Addr: hex.EncodeToString(h)}
	}

	macStr := hex.EncodeToString(h)
	err := e.EncodeElement(macStr, start)
	return errors.Wrap(err, "Encoding MAC address")
}

func (h MacAddress) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(h) != MacAddressSize {
		return xml.Attr{}, &net.AddrError{Err: "invalid MAC address", Addr: hex.EncodeToString(h)}
	}

	macStr := hex.EncodeToString(h)
	attr := xml.Attr{
		Name:  name,
		Value: macStr,
	}

	return attr, nil
}
