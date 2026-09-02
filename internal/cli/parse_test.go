package cli

import (
	"anodik/internal/jobs"
	"flag"
	"testing"
)

func TestResolvePositionalExample(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	options := &jobs.ExampleOptions{}
	parseExampleOptions(fs, options)

	if err := fs.Parse([]string{"rndis"}); err != nil {
		t.Fatal(err)
	}
	if err := resolvePositionalExample(fs, options); err != nil {
		t.Fatal(err)
	}

	if options.Example != "rndis" {
		t.Errorf("Example = %q, want rndis", options.Example)
	}
}

func TestResolvePositionalExampleExplicitFlagWins(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	options := &jobs.ExampleOptions{}
	parseExampleOptions(fs, options)

	if err := fs.Parse([]string{"--example", "rndis"}); err != nil {
		t.Fatal(err)
	}
	if err := resolvePositionalExample(fs, options); err != nil {
		t.Fatal(err)
	}

	if options.Example != "rndis" {
		t.Errorf("Example = %q, want rndis", options.Example)
	}
}

func TestResolvePositionalExampleTooMany(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	options := &jobs.ExampleOptions{}
	parseExampleOptions(fs, options)

	if err := fs.Parse([]string{"rndis", "extra"}); err != nil {
		t.Fatal(err)
	}
	if err := resolvePositionalExample(fs, options); err == nil {
		t.Error("expected an error for more than one leftover positional argument")
	}
}

func TestResolvePositionalExampleNone(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	options := &jobs.ExampleOptions{}
	parseExampleOptions(fs, options)

	if err := fs.Parse([]string{"--all"}); err != nil {
		t.Fatal(err)
	}
	if err := resolvePositionalExample(fs, options); err != nil {
		t.Fatal(err)
	}

	if options.Example != "" {
		t.Errorf("Example = %q, want empty", options.Example)
	}
}
