package version

import "fmt"

var (
	Version   = "1.5.5.1"
	BuildHash = "unknown hash" // will insert in build time
	BuildTime = "unknown time" // will insert in build time
)

func PrintVersion() {
	fmt.Println("WindSend-Relay", "v"+Version)
	fmt.Println("BuildTime:", BuildTime)
	fmt.Println("BuildHash:", BuildHash)
}
