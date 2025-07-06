package main

import "github.com/ridoystarlord/migrato/cmd"

var version = "v1.1.8"

func main() {
	cmd.Version = version
	cmd.Execute()
}
