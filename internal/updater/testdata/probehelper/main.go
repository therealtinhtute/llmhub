// probehelper is a minimal stand-in for the real server binary, used to test
// the staged-candidate probe. It mirrors cmd/server's `version --short`
// contract: print buildinfo.Version to stdout and exit 0. It exits nonzero
// when the environment is not sanitized or the working directory is not
// isolated, so probe tests can prove those properties, and it makes no
// runtime-configuration, database, or service calls.
package main

import (
	"fmt"
	"os"

	"github.com/therealtinhtute/llmhub/internal/buildinfo"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "version" || os.Args[2] != "--short" {
		os.Exit(2)
	}
	if os.Getenv("PGSTORE_DSN") != "" || os.Getenv("LLMHUB_INIT_CONFIG_YAML") != "" {
		fmt.Fprintln(os.Stderr, "runtime config env leaked into probe")
		os.Exit(7)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		fmt.Fprintln(os.Stderr, "probe working directory not isolated")
		os.Exit(8)
	}
	fmt.Println(buildinfo.Version)
}
