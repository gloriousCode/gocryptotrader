package log

// observerSnapshot is immutable so logging with no observers needs only one atomic load.
type observerSnapshot struct {
	callbacks []Observer
}
