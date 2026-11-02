package netwatch

import "context"

// Changes delivers a tick whenever the OS reports a network change (an address,
// interface or route came or went). Bursts are coalesced into one tick. Polling
// every 500 ms stays as a fallback, so an OS that reports nothing only costs speed.
func Changes(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{}, 1)
	notify := func() {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	watch(ctx, notify)
	return ch
}
