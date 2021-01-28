package main

// Storage *
type Storage interface {
	Shorten(url string, exp int64) (string, error)
	ShortlinkInfo(eid string) (string, error)
	Unshorten(eid string) (string, error)
}
