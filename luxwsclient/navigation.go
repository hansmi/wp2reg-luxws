package luxwsclient

import (
	"encoding/xml"
)

func findNavItemByName(name string, items []NavItem) *NavItem {
	for _, item := range items {
		if item.Name == name {
			return &item
		}

		if found := findNavItemByName(name, item.Items); found != nil {
			return found
		}
	}

	return nil
}

// NavRoot represents the navigation structure of a LuxWS server.
type NavRoot struct {
	XMLName xml.Name
	ID      string    `xml:"id,attr"`
	Items   []NavItem `xml:"item"`
}

// FindByName iterates through all items and finds the first with a given name.
// Returns nil if none is found.
func (r *NavRoot) FindByName(name string) *NavItem {
	return findNavItemByName(name, r.Items)
}

// NavItem is an individual entry in the navigation structure.
type NavItem struct {
	ID    string    `xml:"id,attr"`
	Name  string    `xml:"name"`
	Items []NavItem `xml:"item"`
}
