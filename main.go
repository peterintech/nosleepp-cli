package main

import (
	"errors"
	"fmt"
	"os"

	"nosleep/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var exitErr cmd.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Message != "" {
				fmt.Fprintln(os.Stderr, exitErr.Message)
			}
			os.Exit(exitErr.Code)
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
