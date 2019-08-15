//
// Copyright (C) 2019 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"gotesla"
	"net/http"
	"os"
)

func main() {
	var verbose = false

	// Command-line arguments
	var email = flag.String("email", "", "MyTesla email address")
	var password = flag.String("password", "", "MyTesla account password")
	var refresh = flag.Bool("refresh", false, "Refresh existing cached token")
	var jsonOutput = flag.Bool("json", false, "Print token JSON")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	// Parse command-line arguments
	flag.Parse()

	// Don't verify TLS certs...
	tls := &tls.Config{InsecureSkipVerify: true}

	// Get TLS transport
	tr := &http.Transport{TLSClientConfig: tls}

	// Make an HTTPS client
	client := &http.Client{Transport: tr}

	var t *gotesla.Token
	var err error

	// We either are doing a refresh (where refresh == true) or
	// we're doing a fresh login and we need a username and password
	if *refresh {
		var t0 *gotesla.Token
		t0, err = gotesla.LoadCachedToken()
		if err != nil {
			fmt.Println(err)
			return
		}
		t, err = gotesla.RefreshAndCacheToken(client, t0)
	} else if len(*email) > 0 && len(*password) > 0 {

		// Get an authentication token
		t, err = gotesla.GetAndCacheToken(client, email, password)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("Either -refresh needs to be set, or furnish both -email and -password")
		return
	}

	// Output just the token, or the entire JSON structure as appropriate
	if *jsonOutput {
		b, err := json.MarshalIndent(*t, "", "    ")
		if err != nil {
			fmt.Println(err)
		}
		os.Stdout.Write(b)
	} else {
		fmt.Printf("%s\n", t.AccessToken)
	}
}
