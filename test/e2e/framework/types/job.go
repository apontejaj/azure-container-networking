package types

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Job struct {
	t      *testing.T
	Values *JobValues
	Steps  []*StepWrapper
}

type StepWrapper struct {
	Step Step
	Opts *StepOptions
}

func responseDivider(jobname string) {
	totalWidth := 100
	start := 20
	i := 0
	for ; i < start; i++ {
		fmt.Print("#")
	}
	mid := fmt.Sprintf(" %s ", jobname)
	fmt.Print(mid)
	for ; i < totalWidth-(start+len(mid)); i++ {
		fmt.Print("#")
	}
	fmt.Println()
}

func NewJob(t *testing.T) *Job {
	return &Job{
		t: t,
		Values: &JobValues{
			kv: make(map[string]string),
		},
	}
}

func (j *Job) Run() {
	if j.t.Failed() {
		return
	}

	for _, wrapper := range j.Steps {
		err := wrapper.Step.Prevalidate()
		if err != nil {
			require.NoError(j.t, err)
		}
	}

	for _, wrapper := range j.Steps {
		responseDivider(reflect.TypeOf(wrapper.Step).Elem().Name())
		log.Printf("INFO: step options provided: %+v\n", wrapper.Opts)
		err := wrapper.Step.Run()
		if wrapper.Opts.ExpectError {
			require.Error(j.t, err)
		} else {
			require.NoError(j.t, err)
		}
	}

	for _, wrapper := range j.Steps {
		err := wrapper.Step.Postvalidate()
		if err != nil {
			require.NoError(j.t, err)
		}
	}
}

func (j *Job) AddScenario(steps ...StepWrapper) {
	for _, step := range steps {
		j.AddStep(step.Step, step.Opts)
	}
}

func (j *Job) AddStep(step Step, opts *StepOptions) {
	stepName := reflect.TypeOf(step).Elem().Name()
	val := reflect.ValueOf(step).Elem()

	// set default options if none are provided
	if opts == nil {
		opts = &DefaultOpts
	}

	for i, f := range reflect.VisibleFields(val.Type()) {

		// skip saving unexported fields
		if !f.IsExported() {
			continue
		}

		k := reflect.Indirect(val.Field(i)).Kind()

		if k == reflect.String {
			parameter := val.Type().Field(i).Name
			value := val.Field(i).Interface().(string)
			storedValue := j.Values.Get(parameter)

			if storedValue == "" {
				if value != "" {
					if opts.SaveParametersToJob {
						fmt.Printf("%s setting parameter %s in job context to %s\n", stepName, parameter, value)
						j.Values.Set(parameter, value)
					}
					continue
				}
				assert.FailNowf(j.t, "missing parameter", "parameter %s is required for step %s", parameter, stepName)

			}

			if value != "" {
				assert.FailNowf(j.t, "parameter already set", "parameter %s for step %s is already set from previous step", parameter, stepName)
			}

			// don't use log format since this is technically preexecution and easier to read
			fmt.Println(stepName, "using previously stored value for parameter", parameter, "set as", j.Values.Get(parameter))
			val.Field(i).SetString(storedValue)
		}
	}

	j.Steps = append(j.Steps, &StepWrapper{
		Step: step,
		Opts: opts,
	})
}
