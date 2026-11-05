//go:build !darwin

package notify

func send(_ Message) error { return ErrUnsupported }
