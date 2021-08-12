package main

import (
	"github.com/streamingfast/sparkle/cli"
	_ "github.com/streamingfast/sparkle/entity"
	"github.com/streamingfast/sparkle/subgraph"
	"github.com/streamingfast/sushi-generated-priv/exchange"
)

func main() {
	subgraph.MainSubgraphDef = exchange.Definition
	cli.Execute()
}
