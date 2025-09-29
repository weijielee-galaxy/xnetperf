package main

import (
	"fmt"
	"xnetperf/stream"
)

func main() {
	fmt.Println("Testing IP discovery function...")

	// Test with localhost and loopback interface
	ip, err := stream.GetHostIPForHCA("localhost", "test", "lo")
	if err != nil {
		fmt.Printf("Error with 'lo' interface: %v\n", err)
	} else {
		fmt.Printf("Found IP for 'lo' interface: %s\n", ip)
	}

	// Test with empty interface (should try defaults)
	ip2, err2 := stream.GetHostIPForHCA("localhost", "test", "")
	if err2 != nil {
		fmt.Printf("Error with default interfaces: %v\n", err2)
	} else {
		fmt.Printf("Found IP with default interface discovery: %s\n", ip2)
	}
}
