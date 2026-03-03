package hyproxia

func startupMessage(addr string, TLS ...bool) {
	if len(TLS) > 0 && TLS[0] {
		println("Hyproxia v" + version + " running at " + addr + " with TLS")
	} else {
		println("Hyproxia v" + version + " running at " + addr)
	}
}
