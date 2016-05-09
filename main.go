package main

import (
	"fmt"
	"github.com/docopt/docopt.go"
	jibi "github.com/kbatten/jibi/jibi-lib"
)

func main() {
	doc := `usage: jibi [options] <rom>
dev options:
  --dev-status    show 1 second status
  --dev-norender  disable rendering
  --dev-nokeypad  disable keypad input
  --dev-quick     run a quick test cycle
  --dev-nosquash  only display upper left
  --dev-every     print every exectuted instruction`
	args, _ := docopt.Parse(doc, nil, true, "", false)

	rom, err := jibi.ReadRomFile(args["<rom>"].(string))
	if err != nil {
		fmt.Println(err)
		return
	}

	options := jibi.Options{
		Status: args["--dev-status"].(bool),
		Render: !args["--dev-norender"].(bool),
		Keypad: !args["--dev-nokeypad"].(bool),
		Quick:  args["--dev-quick"].(bool),
		Squash: !args["--dev-nosquash"].(bool),
		Every:  args["--dev-every"].(bool),
	}
	gameboy := jibi.New(rom, options)

	gameboy.Run()
}
