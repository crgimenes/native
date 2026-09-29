//go:build !darwin && !linux

package alert

func show(_ Options) (Result, error) { return Result{}, ErrUnsupported }
