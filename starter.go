package webStarter

import (
	"github.com/dennesshen/photon-core-starter/core"
	"photon-web-starter/server"
)

func init() {
	core.RegisterAddModule(server.RunServer)
	core.RegisterShutdownAddModule(server.ShutdownServer)
}
