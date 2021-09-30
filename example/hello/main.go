package main

import (
	"github.com/windzhu0514/shiba/example/hello/hello"
	"github.com/windzhu0514/shiba/shiba"
)

func main() {
	shiba.RegisterModule(1, &hello.Hello{})
	shiba.Start(shiba.WithPprof(), shiba.WithCron())
}
