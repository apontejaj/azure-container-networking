package types

type Step interface {
	Run(values *JobValues) error
	Prevalidate(values *JobValues) error
	DryRun(values *JobValues) error
}
