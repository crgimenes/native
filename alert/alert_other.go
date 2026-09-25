//go:build !darwin

package alert

func show(_ Options) (Result, error) { return Result{}, ErrUnsupported }
