package main

import (
	"deepseek-openrouter-proxy/pkg"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	conf := &pkg.Config{}
	conf.MarginWithENV()

	pkg.Log.Debug("show config detail:")
	pkg.Log.Debug(conf.ToJSON())

	service := pkg.NewHttpService(conf)
	service.Start()
}
