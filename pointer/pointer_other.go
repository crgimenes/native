//go:build !darwin

package pointer

func watch(_ func(Event)) (func(), error) { return nil, ErrUnsupported }
