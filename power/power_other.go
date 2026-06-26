//go:build !darwin && !windows

// Every platform without a backend (Linux included). Keeping the system awake on
// Linux means an org.freedesktop.login1 inhibitor over D-Bus, which would pull in
// a D-Bus dependency this module avoids, or a fragile hand-rolled D-Bus client;
// either way it is out of scope here, so PreventSleep reports ErrUnsupported.

package power

type handle struct{}

func preventSleep(_ string) (handle, error) {
	return handle{}, ErrUnsupported
}

func release(_ handle) error {
	return ErrUnsupported
}
