package metrics

import "github.com/Azure/azure-container-networking/npm/util"

func RecordListEndpointsLatency(timer *Timer) {
	if util.IsWindowsDP() {
		listEndpointsLatency.Observe(timer.timeElapsed())
	}
}

func IncListEndpointsFailures() {
	if util.IsWindowsDP() {
		listEndpointsFailures.Inc()
	}
}

func RecordGetEndpointLatency(timer *Timer) {
	if util.IsWindowsDP() {
		getEndpointLatency.Observe(timer.timeElapsed())
	}
}

func IncGetEndpointFailures() {
	if util.IsWindowsDP() {
		getEndpointFailures.Inc()
	}
}
