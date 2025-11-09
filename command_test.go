package command_test

import (
	"testing"

	"github.com/gloo-foo/testable/assertion"
	"github.com/gloo-foo/testable/run"
	command "github.com/yupsh/paste"
)

func TestPaste_Stdin(t *testing.T) {
	result := run.Command(command.Paste()).
		WithStdinLines("a", "b", "c").Run()
	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 3)
}

func TestPaste_Serial(t *testing.T) {
	result := run.Command(command.Paste(command.Serial)).
		WithStdinLines("a", "b", "c").Run()
	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 1)
}

func TestPaste_CustomDelimiter(t *testing.T) {
	result := run.Command(command.Paste(command.Delimiter(","))).
		WithStdinLines("a", "b").Run()
	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 2)
}

func TestPaste_EmptyInput(t *testing.T) {
	result := run.Quick(command.Paste())
	assertion.NoError(t, result.Err)
	assertion.Empty(t, result.Stdout)
}

