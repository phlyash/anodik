package jobs

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// FlagHelp describes a single CLI flag for help output.
type FlagHelp struct {
	Name string // without leading dashes, e.g. "soc"
	Arg  string // value placeholder, e.g. "<name>"; empty for boolean flags
	Desc string
}

// HelpInfo is a command's self-description: a one-line summary (shown in
// "anodik help") plus its flags (shown in "anodik <command> help"). Jobs
// build it by combining Help() from each embedded Options group, so a flag
// like --soc is documented once in its Options type and every command that
// embeds that type picks up the description automatically.
type HelpInfo struct {
	Summary string
	Usage   string
	Flags   []FlagHelp
}

// Merge combines this HelpInfo's flags with other Options groups' flags,
// keeping Summary/Usage from the receiver. Used by jobs to assemble their
// full help from embedded option groups.
func (h HelpInfo) Merge(others ...HelpInfo) HelpInfo {
	for _, other := range others {
		h.Flags = append(h.Flags, other.Flags...)
	}
	return h
}

// Print writes the command's usage line and flag table to w.
func (h HelpInfo) Print(w io.Writer, command string) {
	fmt.Fprintf(w, "anodik %s — %s\n", command, h.Summary)

	usage := h.Usage
	if usage == "" {
		usage = "[flags]"
	}
	fmt.Fprintf(w, "\nusage: anodik %s %s\n", command, usage)

	if len(h.Flags) == 0 {
		return
	}

	fmt.Fprintln(w, "\nflags:")

	nameWidth := 0
	labels := make([]string, len(h.Flags))
	for i, f := range h.Flags {
		label := "--" + f.Name
		if f.Arg != "" {
			label += " " + f.Arg
		}
		labels[i] = label
		if len(label) > nameWidth {
			nameWidth = len(label)
		}
	}

	for i, f := range h.Flags {
		fmt.Fprintf(w, "  %-*s  %s\n", nameWidth, labels[i], f.Desc)
	}
}

// Describable is implemented by every Job to provide its own help content.
type Describable interface {
	Describe() HelpInfo
}

// CommandHelp is a (name, HelpInfo) pair used to print the global command
// list. Callers (the cli package) build this list once from the same
// registry that dispatches commands, so "anodik help" never drifts from
// what's actually runnable.
type CommandHelp struct {
	Name string
	Info HelpInfo
}

// PrintCommandList writes an alphabetized command summary table to w.
func PrintCommandList(w io.Writer, commands []CommandHelp) {
	sorted := make([]CommandHelp, len(commands))
	copy(sorted, commands)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	nameWidth := 0
	for _, c := range sorted {
		if len(c.Name) > nameWidth {
			nameWidth = len(c.Name)
		}
	}

	fmt.Fprintln(w, "anodik — CMake/OpenOCD/GDB helper for the RISC-V SDK")
	fmt.Fprintln(w, "\nusage: anodik <command> [flags]")
	fmt.Fprintln(w, "\ncommands:")
	for _, c := range sorted {
		fmt.Fprintf(w, "  %-*s  %s\n", nameWidth, c.Name, c.Info.Summary)
	}
	fmt.Fprintln(w, "\nrun 'anodik <command> help' for details on a specific command")
}

// IsHelpToken reports whether arg is a request for help, positionally
// (e.g. "anodik build help") rather than as a --flag.
func IsHelpToken(arg string) bool {
	return strings.EqualFold(arg, "help")
}

// FindHelpToken reports whether any argument in args is a help token, and
// its index. Per the CLI contract, everything from that point on is
// ignored — help wins over any other flag or positional argument.
func FindHelpToken(args []string) (index int, found bool) {
	for i, a := range args {
		if IsHelpToken(a) {
			return i, true
		}
	}
	return -1, false
}
