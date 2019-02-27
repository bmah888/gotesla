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
	"flag"
	"fmt"
	"gotesla"
	"net/http"
	"strconv"
	"strings"
	_ "time"

)

func main() {

	// Command-line arguments
	verbose := flag.Bool("verbose", false, "Verbose output")
	id := flag.String("id", "", "ID of vehicle")

	// Parse command-line arguments
	flag.Parse()

	// Get cached Tesla authentication token
	token, err := gotesla.LoadCachedToken()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Don't verify TLS certs...
	tls := &tls.Config{InsecureSkipVerify: true}

	// Get TLS transport
	tr := &http.Transport{TLSClientConfig: tls}

	// Make an HTTPS client
	client := &http.Client{Transport: tr}

	// Get vehicles list
	vr, err := gotesla.GetVehicles(client, token)
	if err != nil {
		fmt.Println(err)
		return
	}

	if (*verbose) {
		fmt.Printf("%d vehicles retrieved\n", vr.Count)
	}

	// If no Vehicle ID given, so print a list of all the vehicles
	if (*id == "") {
		fmt.Printf("%20s%10s%20s %s\n", "ID", "Model", "VIN", "Name")
		for i := 0; i < vr.Count; i++ {
			
			if (*verbose) {
				fmt.Printf("Option codes: %s\n", vr.Response[i].OptionCodes)
			}
			
			// Parse through option codes to find the vehicle model.
			// This is just a quick 'n' dirty string search.
			// In another part of this program we do a more
			// thorough job of analyzing the option codes.
			var model string = "Unknown"
			if strings.Contains(vr.Response[i].OptionCodes, "MDLS") || strings.Contains(vr.Response[i].OptionCodes, "MS04") {
				model = "Model S"
			} else if strings.Contains(vr.Response[i].OptionCodes, "MDLX") {
				model = "Model X"
			} else if strings.Contains(vr.Response[i].OptionCodes, "MDL3") {
				model = "Model 3"
			}
			
			fmt.Printf("%20d%10s%20s \"%s\"\n", vr.Response[i].Id, model, vr.Response[i].Vin, vr.Response[i].DisplayName)
		}
	}

	// Try to figure out the actual ID
	var idFound int = 0
	iParsed, err := strconv.Atoi(*id)
	for i := 0; i < vr.Count; i++ {
		if vr.Response[i].Id == iParsed || vr.Response[i].Vin == *id || vr.Response[i].DisplayName == *id {
			idFound = vr.Response[i].Id
		}
	}

	if idFound == 0 {
		// Not found
		return
	}

	if *verbose {
		fmt.Printf("Found id %d\n", idFound)
	}

	csr, err := gotesla.GetChargeState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("charge_state: %+v\n", csr)

	clsr, err := gotesla.GetClimateState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("climate_state: %+v\n", clsr)

	dsr, err := gotesla.GetDriveState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("drive_state: %+v\n", dsr)

	mer, err := gotesla.GetMobileEnabled(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("mobile_enabled: %+v\n", mer)


	return
}
