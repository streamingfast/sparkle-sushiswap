# StreamingFast Sushi Generated
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/sushi-generated-priv)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Usagae
```shell
mkdir sushi-generated-priv
cd sushi-generated-priv
mkdir abis
mkdir subgraph
cp /:whatever/sushi-subgraph/abis/* abis
cp /:whatever/sushi-subgraph/subgraphs/* subgraph

sparkle codegen subgraph ./subgraph/subname.yaml github.com/streamingfast/sushi-generated-priv project/subname
go mod tidy
```

To init a database
```shell
    go install ./cmd/subgraph
    subgraph deploy --psql-dsn="postgresql://postgres:@localhost:5432/YOUR_DATABASE?enable_incremental_sort=off&sslmode=disable" project/subgraph
```

## Contributing

**Issues and PR in this repo related strictly to Sushi Generated.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

## License

[Apache 2.0](LICENSE)

