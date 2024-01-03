package types

type Step interface {
	Prevalidate() error
	Run() error
	Postvalidate() error
	ExpectError() bool
	SaveParametersToJob() bool
}
