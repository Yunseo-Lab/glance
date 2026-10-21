// Command dashboard-contracts exports the backend's immutable UI catalog.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/glanceapp/glance/internal/dashboard"
	"os"
)

func main() {
	check := flag.Bool("check", false, "fail if the checked-in catalog is stale")
	flag.Parse()
	data, err := json.MarshalIndent(dashboard.Catalog(), "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	path := "web/src/generated/catalog.json"
	if *check {
		existing, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(existing, data) {
			fmt.Fprintln(os.Stderr, "catalog drift: run npm --prefix web run contracts:generate")
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}
