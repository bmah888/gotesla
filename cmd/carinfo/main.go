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
	vehicles, err := gotesla.GetVehicles(client, token)
	if err != nil {
		fmt.Println(err)
		return
	}

	if *verbose {
		fmt.Printf("%d vehicles retrieved\n", len(*vehicles))
	}

	// If no Vehicle ID given, so print a list of all the vehicles
	if *id == "" {
		fmt.Printf("%20s%10s%20s %s\n", "ID", "Model", "VIN", "Name")
		for i := 0; i < len(*vehicles); i++ {

			if *verbose {
				fmt.Printf("Option codes: %s\n", (*vehicles)[i].OptionCodes)
			}

			// Parse through option codes to find the vehicle model.
			// This is just a quick 'n' dirty string search.
			// In another part of this program we do a more
			// thorough job of analyzing the option codes.
			var model string = "Unknown"
			var optionCodes string = (*vehicles)[i].OptionCodes
			if strings.Contains(optionCodes, "MDLS") || strings.Contains(optionCodes, "MS04") {
				model = "Model S"
			} else if strings.Contains(optionCodes, "MDLX") {
				model = "Model X"
			} else if strings.Contains(optionCodes, "MDL3") {
				model = "Model 3"
			}

			fmt.Printf("%20d%10s%20s \"%s\"\n", (*vehicles)[i].Id, model, (*vehicles)[i].Vin, (*vehicles)[i].DisplayName)
		}
	}

	// Try to figure out the actual ID
	var idFound int = 0
	iParsed, err := strconv.Atoi(*id)
	for i := 0; i < len(*vehicles); i++ {
		if (*vehicles)[i].Id == iParsed || (*vehicles)[i].Vin == *id || (*vehicles)[i].DisplayName == *id {
			idFound = (*vehicles)[i].Id
		}
	}

	if idFound == 0 {
		// Not found
		return
	}

	if *verbose {
		fmt.Printf("Found id %d\n", idFound)
	}

	chs, err := gotesla.GetChargeState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("charge_state: %+v\n", chs)

	cls, err := gotesla.GetClimateState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("climate_state: %+v\n", cls)

	ds, err := gotesla.GetDriveState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("drive_state: %+v\n", ds)

	gs, err := gotesla.GetGuiSettings(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("gui_settings: %+v\n", gs)

	vs, err := gotesla.GetVehicleState(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("vehicle_state: %+v\n", vs)

	vc, err := gotesla.GetVehicleConfig(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("vehicle_config: %+v\n", vc)

	mobileEnabled, err := gotesla.GetMobileEnabled(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("mobile_enabled: %+v\n", mobileEnabled)

	vehicleData, err := gotesla.GetVehicleData(client, token, idFound)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("vehicle_data: %+v\n", vehicleData)

	return
}
