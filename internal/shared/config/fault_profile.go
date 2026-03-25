package config

import "fmt"

type FaultProfile string

const (
	FaultProfileNone                    FaultProfile = "none"
	FaultProfilePaymentTimeout          FaultProfile = "payment_timeout"
	FaultProfilePaymentFlakyFirst       FaultProfile = "payment_flaky_first"
	FaultProfileDuplicateBillingRequest FaultProfile = "duplicate_billing_requested"
	FaultProfileEventConsumerFailure    FaultProfile = "event_consumer_failure"
	FaultProfileOutboxAppendFailure     FaultProfile = "outbox_append_failure"
)

func ParseFaultProfile(value string) (FaultProfile, error) {
	profile := FaultProfile(value)

	switch profile {
	case FaultProfileNone,
		FaultProfilePaymentTimeout,
		FaultProfilePaymentFlakyFirst,
		FaultProfileDuplicateBillingRequest,
		FaultProfileEventConsumerFailure,
		FaultProfileOutboxAppendFailure:
		return profile, nil
	default:
		return "", fmt.Errorf("ATLAS_FAULT_PROFILE must be one of none, payment_timeout, payment_flaky_first, duplicate_billing_requested, event_consumer_failure or outbox_append_failure")
	}
}
