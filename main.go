package main

import "github.com/ridoystarlord/migrato/cmd"

var version = "v1.0.0"

func main() {
	cmd.Version = version
	cmd.Execute()
}
