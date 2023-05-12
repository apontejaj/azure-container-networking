package metrics

import (
	"github.com/Azure/azure-container-networking/npm/util"
	"github.com/prometheus/client_golang/prometheus"
)

func RecordSetPolicyLatency(timer *Timer, op OperationKind, isNested bool) {
	if util.IsWindowsDP() {
		nested := "false"
		if isNested {
			nested = "true"
		}
		labels := prometheus.Labels{
			operationLabel: string(op),
			isNestedLabel:  nested,
		}
		setPolicyLatency.With(labels).Observe(timer.timeElapsed())
	}
}

func IncSetPolicyFailures(op OperationKind, isNested bool) {
	if util.IsWindowsDP() {
		nested := "false"
		if isNested {
			nested = "true"
		}
		labels := prometheus.Labels{
			operationLabel: string(op),
			isNestedLabel:  nested,
		}
		setPolicyFailures.With(labels).Inc()
	}
}

func RecordGetNetworkLatency(timer *Timer) {
	if util.IsWindowsDP() {
		getNetworkLatency.Observe(timer.timeElapsed())
	}
}

func IncGetNetworkFailures() {
	if util.IsWindowsDP() {
		getNetworkFailures.Inc()
	}
}
