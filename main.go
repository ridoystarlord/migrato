package main

import "github.com/ridoystarlord/migrato/cmd"

var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}
