//go:build mage

package main

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Test mg.Namespace

// Unit runs unit tests for the repository.
func (Test) Unit() error {
	mg.Deps(Init)
	log(slog.LevelInfo, "Running unit tests")

	args := []string{
		"test",
		"--platforms", "@io_bazel_rules_go//go/toolchain:linux_amd64",
		"--build_tag_filters=-oci",
	}

	args = append(args, "//...")

	err := sh.Run("bazel", args...)
	if err != nil {
		return err
	}

	return nil
}

type Coverage mg.Namespace

// Run runs unit tests for the repository with code coverage enabled.
func (Coverage) Run() error {
	mg.Deps(Init)
	log(slog.LevelInfo, "Running unit tests with coverage")

	args := []string{"coverage",
		"--platforms", "@io_bazel_rules_go//go/toolchain:linux_" + hostArch(),
		"--build_tag_filters=-oci",
		"//...",
	}

	err := sh.Run("bazel", args...)
	if err != nil {
		return fmt.Errorf("error running unit tests: %w", err)
	}

	return nil
}

// View views the code coverage for this project.
// Information on coverage can be viewed with the lcov genhtml tool (i.e. `brew install lcov`).
func (Coverage) View() error {
	mg.Deps(Coverage.Run)

	// Source: https://bazel.build/configure/coverage#viewing_coverage
	err := sh.Run(
		"genhtml",
		"--branch-coverage",
		"--output",
		"genhtml",
		filepath.Join(outputDirectory(), "_coverage", "_coverage_report.dat"),
	)
	if err != nil {
		return fmt.Errorf("unable to generate HTML coverage report: %w", err)
	}

	return sh.Run("open", filepath.Join("genhtml", "index.html"))
}
