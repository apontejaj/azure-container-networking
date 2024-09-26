package multiflight_test

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/multiflight"
	"github.com/google/go-cmp/cmp"
)

type MockDelayer struct {
	IncreaseF func(time.Time) time.Time
	DecreaseF func(time.Time) time.Time
}

func (m *MockDelayer) Increase(t time.Time) time.Time {
	return m.IncreaseF(t)
}

func (m *MockDelayer) Decrease(t time.Time) time.Time {
	return m.DecreaseF(t)
}

type EqualInt int

func (e EqualInt) Equal(other EqualInt) bool {
	return int(e) == int(other)
}

func TestRefresher(t *testing.T) {
	tests := []struct {
		name      string
		generator func(*int) func(context.Context) (EqualInt, error)
		delayer   multiflight.Delayer
		exp       []int
		numCalls  int
	}{
		{
			"once",
			func(called *int) func(context.Context) (EqualInt, error) {
				return func(context.Context) (EqualInt, error) {
					defer func() { (*called)++ }()
					return EqualInt(42), nil
				}
			},
			&MockDelayer{
				IncreaseF: func(t time.Time) time.Time {
					return t.Add(1 * time.Hour)
				},
				DecreaseF: func(t time.Time) time.Time {
					return t
				},
			},
			[]int{42},
			1,
		},
		{
			"twice",
			func(called *int) func(context.Context) (EqualInt, error) {
				return func(context.Context) (EqualInt, error) {
					defer func() { (*called)++ }()
					return EqualInt(42), nil
				}
			},
			&MockDelayer{
				IncreaseF: func(t time.Time) time.Time {
					return t.Add(1 * time.Hour)
				},
				DecreaseF: func(t time.Time) time.Time {
					return t
				},
			},
			[]int{42, 42},
			2,
		},
		{
			"changes",
			func(called *int) func(context.Context) (EqualInt, error) {
				vals := []int{4, 8, 15, 16, 23, 42}
				return func(context.Context) (EqualInt, error) {
					defer func() { (*called)++ }()
					return EqualInt(vals[*called]), nil
				}
			},
			&MockDelayer{
				IncreaseF: func(t time.Time) time.Time {
					return t.Add(1 * time.Hour)
				},
				DecreaseF: func(t time.Time) time.Time {
					return t
				},
			},
			[]int{4, 8, 15, 16, 23, 42},
			6,
		},
		{
			"cache",
			func(called *int) func(context.Context) (EqualInt, error) {
				vals := []int{4, 4, 15, 16, 23, 42}
				return func(context.Context) (EqualInt, error) {
					defer func() { (*called)++ }()
					return EqualInt(vals[*called]), nil
				}
			},
			&MockDelayer{
				IncreaseF: func(t time.Time) time.Time {
					return t.Add(1 * time.Hour)
				},
				DecreaseF: func(t time.Time) time.Time {
					return t
				},
			},
			[]int{4, 4, 4},
			2,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := 0
			rf := multiflight.NewRefresher(test.generator(&called))
			rf.Delayer = test.delayer

			got := make([]int, len(test.exp))

			for idx, _ := range test.exp {
				gotVal, err := rf.Get(context.TODO())
				if err != nil {
					t.Fatal("unexpected error getting value: err:", err)
				}

				got[idx] = int(gotVal)
			}

			if !cmp.Equal(test.exp, got) {
				t.Error("received values differ from expected: diff:", cmp.Diff(test.exp, got))
			}

			if test.numCalls != called {
				t.Error("unexpected number of calls: exp:", test.numCalls, "got:", called)
			}
		})
	}
}
