package mechutil

import (
	"testing"
)

func TestRoutingTable(t *testing.T) {
	var table RoutingTable

	table.Populate(RouteEntry{})
	table.Populate(RouteEntry{})

	if len(table.routes) != 2 {
		t.Fatalf("Failed to append a routes")
	}
}
