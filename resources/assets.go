package resources

import _ "embed"

//go:embed server.crt
var DefaultCertPEM []byte

//go:embed server.key
var DefaultKeyPEM []byte

//go:embed id_ed25519
var DefaultHostKeyPEM []byte

//go:embed id_rsa
var DefaultRSAHostKeyPEM []byte
