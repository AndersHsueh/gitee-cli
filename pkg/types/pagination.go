package types

// PageLinks represents pagination links from HTTP Link header.
type PageLinks struct {
	Next  string
	Prev  string
	Last  string
	First string
}
