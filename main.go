package main

import "github.com/ridoystarlord/migrato/cmd"

var version = "v1.1.6"

func main() {
	cmd.Version = version
	cmd.Execute()
}
