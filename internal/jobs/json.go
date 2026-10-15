package jobs

import (
	"encoding/json"
	"io"
	"os"
)

var jsonOut io.Writer = os.Stdout

type JSONOutput struct {
	JSON bool
}

func (JSONOutput) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "json", Desc: "print machine-readable JSON instead of text"},
		},
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(jsonOut)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
