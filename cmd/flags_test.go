package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetFlagBool(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		cmd.Flags().Bool("test", true, "")
		v, err := getFlagBool(cmd, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v {
			t.Errorf("got %v, want true", v)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		_, err := getFlagBool(cmd, "no-such-flag")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetFlagInt(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		cmd.Flags().Int("jobs", 4, "")
		v, err := getFlagInt(cmd, "jobs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 4 {
			t.Errorf("got %d, want 4", v)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		_, err := getFlagInt(cmd, "no-such-flag")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetFlagString(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		cmd.Flags().String("input", "foo.conf", "")
		v, err := getFlagString(cmd, "input")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "foo.conf" {
			t.Errorf("got %q, want %q", v, "foo.conf")
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		_, err := getFlagString(cmd, "no-such-flag")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetFlagStringSlice(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		cmd.Flags().StringSlice("exclude", []string{"a", "b"}, "")
		v, err := getFlagStringSlice(cmd, "exclude")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(v) != 2 || v[0] != "a" || v[1] != "b" {
			t.Errorf("got %v, want [a b]", v)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		_, err := getFlagStringSlice(cmd, "no-such-flag")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
