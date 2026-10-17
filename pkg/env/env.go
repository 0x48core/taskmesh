// Package env reports the current environment (dev, test, stg, prod).
package env

import (
	"fmt"
	"os"
)

// Key is the environment variable that selects the environment.
const Key = "APP_ENV"

const (
	Dev  = "dev"
	Test = "test"
	Stg  = "stg"
	Prod = "prod"
)

// Current returns the environment named by APP_ENV, or Dev when it is unset.
// It returns an error for any value other than dev, test, stg or prod.
func Current() (string, error) {
	e := os.Getenv(Key)
	if e == "" {
		return Dev, nil
	}
	switch e {
	case Dev, Test, Stg, Prod:
		return e, nil
	default:
		return "", fmt.Errorf("env: unknown %s %q (want dev, test, stg or prod)", Key, e)
	}
}
