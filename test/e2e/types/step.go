package types

type Step interface {
	Prevalidate(values *JobValues) error
	Run(values *JobValues) error
	Postvalidate(values *JobValues) error
	ExpectError() bool
	SaveParametersToJob() bool
}
