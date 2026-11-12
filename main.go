// Command pretty-alias shows the aliases from your shell config in a
// scrollable, syntax-highlighted table. All the logic lives in internal/.
//
// This file stays in the module root (not cmd/) so that
// `go install github.com/sharon-xa/pretty-alias/v2@latest` keeps working.
package main

import (
	"os"

	"github.com/sharon-xa/pretty-alias/v2/internal/app"
)

// version is set at build time: go build -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
