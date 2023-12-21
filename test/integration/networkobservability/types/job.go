package types

import (
	"fmt"
	"reflect"
)

type Job struct {
	Values *JobValues
	Steps  []Step
}

func NewJob() *Job {
	return &Job{
		Values: &JobValues{},
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
	for _, step := range j.Steps {
		err := step.Run(j.Values)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *Job) AddStep(step Step) {
	val := reflect.ValueOf(step).Elem()
	for i := 0; i < val.NumField(); i++ {
		if reflect.TypeOf(val.Type().Field(i)).Kind() == reflect.String {
			parameter := val.Type().Field(i).Name
			value := val.Field(i).Interface()
			j.Values.Set(parameter, value.(string))
		}
	}
}
