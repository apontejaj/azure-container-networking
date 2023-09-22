//go:build connection

package connection

type nodeInfo struct {
	ip             string
	desiredNCCount int
	allocatedNCs   []ncInfo
}

type vnetInfo struct {
}

type ncInfo struct {
	NCID         string
	PodName      string
	PodNamespace string
}
