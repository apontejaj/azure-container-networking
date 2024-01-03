package types

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Job struct {
	t      *testing.T
	Values *JobValues
	Steps  []Step
}

func responseDivider(jobname string) {
	fmt.Println("################## " + jobname + " ##################")
}

func NewJob(t *testing.T) *Job {
	return &Job{
		t: t,
		Values: &JobValues{
			kv: make(map[string]string),
		},
	}
}

func (j *Job) Validate() error {
	// ensure that each property in the
	for _, step := range j.Steps {
		val := reflect.ValueOf(step).Elem()
		for i := 0; i < val.NumField(); i++ {
			fmt.Println(val.Type().Field(i).Name)
		}
	}
	return nil
}

func (j *Job) Run() error {
	if j.t.Failed() {
		return fmt.Errorf("job failed")
	}

	for _, step := range j.Steps {
		err := step.Prevalidate(j.Values)
		if err != nil {
			assert.NoError(j.t, err)
			return err
		}
	}

	for _, step := range j.Steps {
		responseDivider(reflect.TypeOf(step).Elem().Name())
		err := step.Run(j.Values)
		if err != nil {
			assert.NoError(j.t, err)
			return err
		}
	}

	for _, step := range j.Steps {
		err := step.Postvalidate(j.Values)
		if err != nil {
			assert.NoError(j.t, err)
			return err
		}
	}
	return nil
}

func (j *Job) AddStep(step Step) {
	stepName := reflect.TypeOf(step).Elem().Name()
	val := reflect.ValueOf(step).Elem()

	// skip saving parameters to job
	if !step.SaveParametersToJob() {
		return
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
					j.Values.Set(parameter, value)
					continue
				} else {
					assert.FailNow(j.t, "parameter "+parameter+" is required for step "+stepName)
				}
			}

			if value != "" {
				assert.FailNow(j.t, "parameter "+parameter+" for step "+stepName+" is already set from previous step ")
				j.t.Fatal("parameter", parameter, "for step", stepName, "is already set from previous step")
				panic("parameter " + parameter + " for step " + stepName + " is already set from previous step")
			}

			fmt.Println(stepName, "using previously stored value for parameter", parameter, "set as", j.Values.Get(parameter))
			val.Field(i).SetString(storedValue)
		}
	}

	j.Steps = append(j.Steps, step)

}
