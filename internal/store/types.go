package store

// Item represents a key-value entry in the store.
// Expiry is a Unix timestamp; 0 means no expiry.
type Item struct {
	Value  string
	Expiry int64
}
