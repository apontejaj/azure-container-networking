package retry

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func createFunctionWithFailurePattern(errorPattern []error) func() error {
	s := 0
	return func() error {
		if s >= len(errorPattern) {
			return nil
		}
		result := errorPattern[s]
		s++
		return result
	}
}

func TestRunWithRetries(t *testing.T) {
	errMock := errors.New("mock error")
	runs := 4

	tests := []struct {
		name    string
		wantErr bool
		f       func() error
	}{
		{
			name:    "Succeed on first try",
			f:       createFunctionWithFailurePattern([]error{}),
			wantErr: false,
		},
		{
			name:    "Succeed on first try do not check again",
			f:       createFunctionWithFailurePattern([]error{nil, errMock, errMock, errMock}),
			wantErr: false,
		},
		{
			name:    "Succeed on last try",
			f:       createFunctionWithFailurePattern([]error{errMock, errMock, errMock, nil, errMock}),
			wantErr: false,
		},
		{
			name:    "Fail after too many attempts",
			f:       createFunctionWithFailurePattern([]error{errMock, errMock, errMock, errMock, nil, nil}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := Do(tt.f, runs, 100)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
