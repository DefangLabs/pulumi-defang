package testutil

import (
	"sync"

	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// ResourceRecord is one resource registration seen by CollectResources.
type ResourceRecord struct {
	Typ    string
	Name   string
	Inputs property.Map
}

// CollectResources returns a mock resource monitor and a pointer to the slice it
// populates, for tests that assert on which resources a Construct created.
//
// NewResourceF runs concurrently (one goroutine per registration), so the append
// is mutex-guarded; read the slice only after Construct returns.
func CollectResources() (*integration.MockResourceMonitor, *[]ResourceRecord) {
	var mu sync.Mutex
	var records []ResourceRecord
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			mu.Lock()
			records = append(records, ResourceRecord{
				Typ:    string(args.TypeToken),
				Name:   args.Name,
				Inputs: args.Inputs,
			})
			mu.Unlock()
			return args.Name, args.Inputs, nil
		},
	}
	return mock, &records
}

// CountType returns how many records match the given type token.
func CountType(records []ResourceRecord, typ string) int {
	return CountTypeWhere(records, typ, func(property.Map) bool { return true })
}

// CountTypeWhere returns how many records match the given type token and predicate.
func CountTypeWhere(records []ResourceRecord, typ string, pred func(property.Map) bool) int {
	n := 0
	for _, r := range records {
		if r.Typ == typ && pred(r.Inputs) {
			n++
		}
	}
	return n
}

// FindTypeWhere returns the first record matching the given type token and
// predicate, or nil.
func FindTypeWhere(records []ResourceRecord, typ string, pred func(property.Map) bool) *ResourceRecord {
	for i := range records {
		if records[i].Typ == typ && pred(records[i].Inputs) {
			return &records[i]
		}
	}
	return nil
}
