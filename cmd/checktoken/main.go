//
// Copyright (C) 2019 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gotesla"
	"os"
)

var jsonOutput = false

// Return true if the cached token is valid, false otherwise
func checkCached() bool {

	// Try to read the cached token. If it doesn't exist,
	// clearly that's invalid.
	t, err := gotesla.LoadCachedToken()
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Check validity
	return gotesla.CheckToken(t)
}

// Print token object in JSON representation
func printCached() {
	t, err := gotesla.LoadCachedToken()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Output just the token, or the entire JSON structure as appropriate
	if jsonOutput {
		b, err := json.MarshalIndent(*t, "", "    ")
		if err != nil {
			fmt.Println(err)
			return
		}
		os.Stdout.Write(b)
	} else {
		fmt.Printf("%s\n", t.AccessToken)
	}

}

// Delete the cached token
func deleteCached() {
	err := gotesla.DeleteCachedToken()
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	var verbose = false

	// Command-line arguments
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&jsonOutput, "json", false, "JSON output")

	// Define new flag.Usage() so we can print the valid commands
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [flags] COMMAND:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Where COMMAND is one of:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    check   Check stored token for validity\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    delete  Delete stored token\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    print   Print stored token\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		flag.PrintDefaults()
	}

	// Parse command-line arguments
	flag.Parse()

	// We need exactly one word after any arguments...it's a command
	if flag.NArg() != 1 {
		fmt.Println("Need exactly one command")
		return
	}

	// Commands are:
	// check, delete, print
	switch flag.Arg(0) {

	// check
	// Check the validity of the cached token
	case "check":
		{
			if checkCached() == false {
				// XXX find a more graceful way to exit
				os.Exit(1)
			}
		}

	case "clear":
		deleteCached()

	// print
	// Print the cached token in JSON representation
	case "print":
		printCached()

	default:
		fmt.Println("Invalid command")

	}

	/*
		// Don't verify TLS certs...
		tls := &tls.Config{InsecureSkipVerify: true}

		// Get TLS transport
		tr := &http.Transport{TLSClientConfig: tls}

		// Make an HTTPS client
		client := &http.Client{Transport: tr}

		// Get an authentication token
		t, err := gotesla.GetAndCacheToken(client, email, password)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Output just the token, or the entire JSON structure as appropriate
		if *jsonOutput {
			b, err := json.Marshal(*t)
			if err != nil {
				fmt.Println(err)
			}
			os.Stdout.Write(b)
		} else {
			fmt.Printf("%s\n", t.AccessToken)
		}
	*/
}
