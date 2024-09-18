package nmagent

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net"
)

type MacAddress net.HardwareAddr

func (h *MacAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var macStr string
	if err := d.DecodeElement(&macStr, &start); err != nil {
		return err
	}

	// Convert the string (without colons) into a valid MacAddress
	mac, err := hex.DecodeString(macStr)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	*h = MacAddress(mac)
	return nil
}

func (h *MacAddress) UnmarshalXMLAttr(attr xml.Attr) error {
	macStr := attr.Value
	mac, err := hex.DecodeString(macStr)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	*h = MacAddress(mac)
	return nil
}

func (h MacAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(h) != 6 {
		return fmt.Errorf("invalid MAC address")
	}

	macStr := hex.EncodeToString(h)
	return e.EncodeElement(macStr, start)
}

func (h MacAddress) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(h) != 6 {
		return xml.Attr{}, fmt.Errorf("invalid MAC address")
	}

	macStr := hex.EncodeToString(h)
	attr := xml.Attr{
		Name:  name,
		Value: macStr,
	}

	return attr, nil
}
