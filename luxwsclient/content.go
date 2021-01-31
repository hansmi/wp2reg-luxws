package luxwsclient

import (
	"encoding/xml"
)

func findContentItemByName(name string, items []ContentItem) *ContentItem {
	for _, i := range items {
		if i.Name == name {
			return &i
		}

		if found := findContentItemByName(name, i.Items); found != nil {
			return found
		}
	}

	return nil
}

// ContentRoot contains all items returned by a GET request to a LuxWS server.
type ContentRoot struct {
	XMLName xml.Name
	Items   []ContentItem `xml:"item"`
}

// FindByName iterates through all items and finds the first with a given name.
// Returns nil if none is found.
func (r *ContentRoot) FindByName(name string) *ContentItem {
	return findContentItemByName(name, r.Items)
}

// ContentItem is an individual entry on a content page.
type ContentItem struct {
	ID      string              `xml:"id,attr"`
	Name    string              `xml:"name"`
	Min     *string             `xml:"min"`
	Max     *string             `xml:"max"`
	Step    *string             `xml:"step"`
	Unit    *string             `xml:"unit"`
	Div     *string             `xml:"div"`
	Raw     *string             `xml:"raw"`
	Value   *string             `xml:"value"`
	Columns []string            `xml:"columns"`
	Headers []string            `xml:"headers"`
	Options []ContentItemOption `xml:"option"`
	Items   []ContentItem       `xml:"item"`
}

// ContentItemOption represents one option among others of a content item.
type ContentItemOption struct {
	Value string `xml:"value,attr"`
	Name  string `xml:",chardata"`
}
