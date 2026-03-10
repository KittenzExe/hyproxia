//go:build !linux

package hyproxia

import "fmt"

func (p *Proxy) ListenPrefork(addr string) error {
	fmt.Println("Prefork is not supported on this platform. Failing back to single-process mode")
	return p.server.ListenAndServe(addr)
}
