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

/* Command-line argument processing */

func main() {
	var verbose bool = false

	/* Seed random number generator, for semi-random polling interval */
	
	/* Command-line arguments */
	var email = flag.String("email", "", "MyTesla email address")
	var password = flag.String("password", "", "MyTesla account password")
	var jsonOutput = flag.Bool("json", false, "Print token JSON")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	/* Parse command-line arguments */
	flag.Parse()
	
	/* Don't verify TLS certs... */
	tls := &tls.Config{InsecureSkipVerify: true}
	
	/* Get TLS transport */
	tr := &http.Transport{TLSClientConfig: tls}
	
	/* Make an HTTPS client */
	client := &http.Client{Transport: tr}
	
	/* Get an authentication token */
	t, err := gotesla.GetToken(client, email, password)
	if err != nil {
		fmt.Println(err)
		return
	}

	/* Output just the token, or the entire JSON structure as appropriate */
	if *jsonOutput {
		b, err := json.Marshal(t)
		if err != nil {
			fmt.Println(err)
		}
		os.Stdout.Write(b)
	} else {
		fmt.Printf("%s\n", t.AccessToken)
	}
}
