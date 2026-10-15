package packetmanager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type QuestionOption int

const (
	Yes QuestionOption = 1 << iota
	No
	Skip
	Invalid = 0
)

func Ask(question string, options QuestionOption) (QuestionOption, error) {
	return askFrom(os.Stdin, os.Stdout, question, options)
}

func askFrom(in io.Reader, out io.Writer, question string, options QuestionOption) (QuestionOption, error) {
	var parts []string
	if options&Yes == Yes {
		parts = append(parts, "Yes(y)")
	}
	if options&No == No {
		parts = append(parts, "No(n)")
	}
	if options&Skip == Skip {
		parts = append(parts, "Skip(s)")
	}

	optStr := fmt.Sprintf("[%s]", strings.Join(parts, "/"))
	reader := bufio.NewReader(in)

	for {
		fmt.Fprintf(out, "%s %s ", question, optStr)

		input, readErr := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y":
			if options&Yes == Yes {
				return Yes, nil
			}
		case "n":
			if options&No == No {
				return No, nil
			}
		case "s":
			if options&Skip == Skip {
				return Skip, nil
			}
		}

		if readErr != nil {
			fmt.Fprintln(out)
			return Invalid, fmt.Errorf("no answer to %q: stdin is closed (run the command in a terminal)", question)
		}
		fmt.Fprintln(out, "Invalid option, please choose from the available options.")
	}
}
