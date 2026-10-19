package apex

// VirtualHost represents a registered port to forward on the local host for a request to the given Domain
type VirtualHost struct {
	// Domain is the Host to parse for the redirect
	Domain string
	// Port is the localhost port forwarded to when Domain is parsed
	Port uint16
}
