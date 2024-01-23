package types

import (
	"fmt"
	"reflect"
)

type Stop struct {
	BackgroundID string
	Step         Step
}

func (c *Stop) Run() error {
	err := c.Step.Stop()
	if err != nil {
		stepName := reflect.TypeOf(c.Step).Elem().Name()
		return fmt.Errorf("failed to stop step: %s with err %w", stepName, err)
	}
	return nil
}

func (c *Stop) Stop() error {
	return nil
}

func (c *Stop) Prevalidate() error {
	return nil
}

func (c *Stop) Postvalidate() error {
	return nil
}
