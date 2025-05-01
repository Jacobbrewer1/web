//go:build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GenerateAll() {
	mg.Deps(Init)
	mg.Deps(mg.F(Generate.mock, false))
}

type Generate mg.Namespace

func (Generate) Mock() error {
	mg.Deps(mg.F(Generate.mock, true))

	// Cover any new dependencies that may have been added by the code generation.
	return VendorDeps()
}

func (Generate) mock(shouldVendor bool) error {
	mg.Deps(Init)

	args := []string{
		"run",
		"//:gen_mock",
	}

	if err := sh.Run("bazel", args...); err != nil {
		return fmt.Errorf("error writing generated mock files: %w", err)
	}

	if shouldVendor {
		if err := VendorDeps(); err != nil {
			return err
		}
	}

	return nil
}
