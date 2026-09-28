package resources

import _ "embed"

//go:embed server.crt
var DefaultCertPEM []byte

//go:embed server.key
var DefaultKeyPEM []byte
