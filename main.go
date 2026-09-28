package main

import (
	"srvx/cmd"
	"srvx/resources"
)

func main() {
	cmd.SetDefaultTLS(resources.DefaultCertPEM, resources.DefaultKeyPEM)
	cmd.Execute()
}
