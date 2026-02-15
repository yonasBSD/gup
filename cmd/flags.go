package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func getFlagBool(cmd *cobra.Command, name string) (bool, error) {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false, fmt.Errorf("can not parse command line argument (--%s): %w", name, err)
	}
	return v, nil
}

func getFlagInt(cmd *cobra.Command, name string) (int, error) {
	v, err := cmd.Flags().GetInt(name)
	if err != nil {
		return 0, fmt.Errorf("can not parse command line argument (--%s): %w", name, err)
	}
	return v, nil
}

func getFlagString(cmd *cobra.Command, name string) (string, error) {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("can not parse command line argument (--%s): %w", name, err)
	}
	return v, nil
}

func getFlagStringSlice(cmd *cobra.Command, name string) ([]string, error) {
	v, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil, fmt.Errorf("can not parse command line argument (--%s): %w", name, err)
	}
	return v, nil
}
