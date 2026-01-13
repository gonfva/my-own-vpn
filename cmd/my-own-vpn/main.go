package main

import (
    "fmt"
    "os"
)

// Version is set during build
var Version = "dev"

func main() {
    fmt.Printf("My Own VPN v%s\n", Version)
    os.Exit(0)
}
