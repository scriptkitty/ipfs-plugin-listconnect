package main

import (
	pl "connlist/listplugin"

	"github.com/ipfs/go-ipfs/plugin"
)

var Plugins = []plugin.Plugin{
	&pl.ConnectPlugin{},
}
