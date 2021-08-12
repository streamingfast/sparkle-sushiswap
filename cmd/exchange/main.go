package main

import (
	"github.com/streamingast/sushi-generated-priv/exchange/exchange"
	"github.com/streamingfast/sparkle/cli"
	"github.com/streamingfast/sparkle/entity"
	"github.com/streamingfast/sparkle/subgraph"
)

func main() {
	subgraph.MainSubgraphDef = exchange.Definition
	cli.Execute()
}
