package luxwslang

import "fmt"

// All returns a slice of all supported terminologies.
func All() (result []*Terminology) {
	return append(result,
		German,
		English,
	)
}

// LookupByID tries to find a terminology with the given ID (e.g. "en").
func LookupByID(id string) (*Terminology, error) {
	for _, cur := range All() {
		if cur.ID == id {
			return cur, nil
		}
	}

	return nil, fmt.Errorf("language %q not found", id)
}
