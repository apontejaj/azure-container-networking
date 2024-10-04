package nmagent

// equalPtr compares two Interfaces objects for equality.
func (i *Interfaces) equalPtr(other *Interfaces) bool {
	if len(i.Entries) != len(other.Entries) {
		return false
	}
	for idx, entry := range i.Entries {
		if !entry.equalPtr(&other.Entries[idx]) {
			return false
		}
	}
	return true
}

// Equal compares two Interfaces objects for equality.
func (i Interfaces) Equal(other Interfaces) bool {
	return i.equalPtr(&other)
}

// equalPtr compares two Interface objects for equality.
func (i *Interface) equalPtr(other *Interface) bool {
	if len(i.InterfaceSubnets) != len(other.InterfaceSubnets) {
		return false
	}
	for idx, subnet := range i.InterfaceSubnets {
		if !subnet.equalPtr(&other.InterfaceSubnets[idx]) {
			return false
		}
	}
	if i.IsPrimary != other.IsPrimary || !i.MacAddress.Equal(other.MacAddress) {
		return false
	}
	return true
}

// Equal compares two Interface objects for equality.
func (i Interface) Equal(other Interface) bool {
	return i.equalPtr(&other)
}

// equalPtr compares two InterfaceSubnet objects for equality.
func (s *InterfaceSubnet) equalPtr(other *InterfaceSubnet) bool {
	if len(s.IPAddress) != len(other.IPAddress) {
		return false
	}
	if s.Prefix != other.Prefix {
		return false
	}
	for idx, ip := range s.IPAddress {
		if !ip.equalPtr(&other.IPAddress[idx]) {
			return false
		}
	}
	return true
}

// Equal compares two InterfaceSubnet objects for equality.
func (s InterfaceSubnet) Equal(other InterfaceSubnet) bool {
	return s.equalPtr(&other)
}

// equalPtr compares two NodeIP objects for equality.
func (ip *NodeIP) equalPtr(other *NodeIP) bool {
	return ip.IsPrimary == other.IsPrimary && ip.Address.Equal(other.Address)
}

// Equal compares two NodeIP objects for equality.
func (ip NodeIP) Equal(other NodeIP) bool {
	return ip.equalPtr(&other)
}
