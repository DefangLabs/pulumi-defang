package main

import (
	"testing"
)

const TEST_PROJECT = "cd-test"

func TestRunVersion(t *testing.T) {
	if run("cd", "--version") != 0 {
		t.Error("run --version flag should exit with 0")
	}
}

func TestRunUsageError(t *testing.T) {
	if run() != 2 {
		t.Error("run should exit with 2 on usage error")
	}
}

func TestRunHelp(t *testing.T) {
	t.Setenv("PROJECT", TEST_PROJECT)
	if run("cd", "help") != 0 {
		t.Error("run help flag should exit with 0")
	}
}
