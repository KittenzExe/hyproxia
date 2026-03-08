//go:build !linux

package hyproxia

import "errors"

func (p *Proxy) ListenPrefork(addr string) error {
	return errors.New("prefork is not supported on this platform... sorry :c")
}
