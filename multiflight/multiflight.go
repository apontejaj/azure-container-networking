package multiflight

import (
	"context"
	"time"
)

type Equalable[T any] interface {
	Equal(other T) bool
}

func NewRefresher[T Equalable[T]](doFunc func(context.Context) (T, error)) *Refresher[T] {
	return &Refresher[T]{
		DoFunc: doFunc,
	}
}

type Delayer interface {
	Increase(time.Time) time.Time
	Decrease(time.Time) time.Time
}

type Refresher[T Equalable[T]] struct {
	DoFunc   func(context.Context) (T, error)
	Delayer  Delayer
	nextCall time.Time
	lastVal  T
}

func (r *Refresher[T]) Get(ctx context.Context) (T, error) {
	if time.Now().After(r.nextCall) {
		nextVal, err := r.DoFunc(ctx)
		if err != nil {
			return *new(T), err
		}

		if r.lastVal.Equal(nextVal) {
			r.nextCall = r.Delayer.Increase(time.Now())
		}

		r.lastVal = nextVal
	}
	return r.lastVal, nil
}
