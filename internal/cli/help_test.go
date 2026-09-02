package cli

import (
	"anodik/internal/jobs"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandTableCoversEveryDispatchableCommand(t *testing.T) {
	// Sanity check that the table isn't accidentally missing an entry —
	// every command must have a working zero-value Describe().
	for name, cmd := range commandTable {
		info := cmd.job().Describe()
		if info.Summary == "" {
			t.Errorf("command %q: Describe().Summary is empty", name)
		}
	}
}

func TestParseArgsHelpToken(t *testing.T) {
	_, err := ParseArgs([]string{"help"}, "", "")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want flag.ErrHelp", err)
	}
}

func TestParseArgsPerCommandHelp(t *testing.T) {
	_, err := ParseArgs([]string{"build", "help"}, "", "")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want flag.ErrHelp", err)
	}
}

func TestParseArgsHelpIgnoresTrailingArgs(t *testing.T) {
	// Per the CLI contract, anything after "help" (valid or not) must not
	// cause a different error — help always wins.
	_, err := ParseArgs([]string{"build", "help", "--nonexistent-flag"}, "", "")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want flag.ErrHelp", err)
	}
}

func TestParseArgsUnknownCommand(t *testing.T) {
	_, err := ParseArgs([]string{"definitely_not_a_command"}, "", "")
	if err == nil || errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want a plain unknown-command error", err)
	}
}

func TestParseArgsFallsBackToBuildForKnownExample(t *testing.T) {
	sdkRoot := t.TempDir()
	exampleDir := filepath.Join(sdkRoot, "examples", "rndis")
	if err := os.MkdirAll(exampleDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmakeLists := "project(rndis LANGUAGES C)\nadd_executable(${PROJECT_NAME} src/main.c)\n"
	if err := os.WriteFile(filepath.Join(exampleDir, "CMakeLists.txt"), []byte(cmakeLists), 0644); err != nil {
		t.Fatal(err)
	}

	workingDir := t.TempDir()

	job, err := ParseArgs([]string{"rndis"}, sdkRoot, workingDir)
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}

	buildJob, ok := job.(*jobs.BuildJob)
	if !ok {
		t.Fatalf("job type = %T, want *jobs.BuildJob", job)
	}
	if buildJob.Example != "rndis" {
		t.Errorf("Example = %q, want rndis", buildJob.Example)
	}
}
