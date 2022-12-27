package subcmds

import (
	_ "embed"
	"fmt"
)

//go:embed full-help.md
var fullHelpText string

func ShowHelp() {
	fmt.Println(fullHelpText)
}
