package cmd

import (
	"context"
	"errors"
	"testing"
)

func Test_browserCommandError(t *testing.T) {
	t.Parallel()

	otherErr := errors.New("some error")
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: nil},
		{name: "other", err: otherErr, want: otherErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := browserCommandError(tt.err)
			if !errors.Is(got, tt.want) || (tt.want == nil && got != nil) {
				t.Fatalf("browserCommandError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
