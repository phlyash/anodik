package packetmanager

import (
	"bufio"
	"fmt"
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

func Ask(question string, options QuestionOption) QuestionOption {
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
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s %s ", question, optStr)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y":
			if options&Yes == Yes {
				return Yes
			}
		case "n":
			if options&No == No {
				return No
			}
		case "s":
			if options&Skip == Skip {
				return Skip
			}
		}

		fmt.Println("Invalid option, please choose from the available options.")
	}
}
