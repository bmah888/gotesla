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

type OptionDecode struct {
	OptionCode, Decode, OptionClass string
}

var decoder = [...]OptionDecode{
	{OptionCode: "MDLS", Decode: "Model S", OptionClass: "model"},
	{OptionCode: "MS03", Decode: "Model S", OptionClass: "model"},
	{OptionCode: "MS04", Decode: "Model S", OptionClass: "model"},
	{OptionCode: "MDLX", Decode: "Model X", OptionClass: "model"},
	{OptionCode: "MDL3", Decode: "Model 3", OptionClass: "model"},

	{OptionCode: "APH0", Decode: "Autopilot 2.0 Hardware", OptionClass: "autopilothw"},
	{OptionCode: "APH2", Decode: "Autopilot 2.0 Hardware", OptionClass: "autopilothw"},
	{OptionCode: "APH3", Decode: "Autopilot 2.5 Hardware", OptionClass: "autopilothw"},
	{OptionCode: "APPA", Decode: "Autopilot 1.0 Hardware", OptionClass: "autopilothw"},
	{OptionCode: "APPB", Decode: "Enhanced Autopilot", OptionClass: "autopilothw"},

	{OptionCode: "BP00", Decode: "No Ludicrous", OptionClass: "ludicrous"},
	{OptionCode: "BP01", Decode: "Ludicrous Speed Upgrade", OptionClass: "ludicrous"},

	{OptionCode: "BR00", Decode: "No battery firmware limit", OptionClass: "batterylimit"},
	{OptionCode: "BR03", Decode: "Battery firmware limit (60 kWh)", OptionClass: "batterylimit"},
	{OptionCode: "BR05", Decode: "Battery firmware limit (60 kWh)", OptionClass: "batterylimit"},

	{OptionCode: "BT37", Decode: "75 kWh", OptionClass: "battery"},
	{OptionCode: "BT40", Decode: "40 kWh", OptionClass: "battery"},
	{OptionCode: "BT60", Decode: "60 kWh", OptionClass: "battery"},
	{OptionCode: "BT70", Decode: "70 kWh", OptionClass: "battery"},
	{OptionCode: "BT85", Decode: "85 kWh", OptionClass: "battery"},
	{OptionCode: "BTX4", Decode: "90 kWh", OptionClass: "battery"},
	{OptionCode: "BTX5", Decode: "75 kWh", OptionClass: "battery"},
	{OptionCode: "BTX6", Decode: "100 kWh", OptionClass: "battery"},
	{OptionCode: "BTX7", Decode: "75 kWh", OptionClass: "battery"},
	{OptionCode: "BTX8", Decode: "85 kWh", OptionClass: "battery"},

	{OptionCode: "CW00", Decode: "No Cold Weather Package", OptionClass: "subzero"},
	{OptionCode: "CW01", Decode: "Cold Weather Package", OptionClass: "subzero"},

	{OptionCode: "DA00", Decode: "No Autopilot", OptionClass: "autopilot"},
	{OptionCode: "DA01", Decode: "Autopilot Active Safety", OptionClass: "autopilot"},
	{OptionCode: "DA02", Decode: "Autopilot Convenience", OptionClass: "autopilot"},
	{OptionCode: "DCF0", Decode: "Autopilot Convenience", OptionClass: "autopilot"},

	{OptionCode: "DRLH", Decode: "Left-Hand Drive", OptionClass: "handedness"},
	{OptionCode: "DRRH", Decode: "Right-Hand Drive", OptionClass: "handedness"},

	{OptionCode: "DV2W", Decode: "RWD", OptionClass: "drivewheels"},
	{OptionCode: "DV4W", Decode: "AWD", OptionClass: "drivewheels"},

	{OptionCode: "TP01", Decode: "Tech Package (no AP)", OptionClass: "tech"},
	{OptionCode: "TP02", Decode: "Tech Package (AP)", OptionClass: "tech"},
	{OptionCode: "TP03", Decode: "Tech Package (EAP)", OptionClass: "tech"},
}

func printOptionCodes(codeString string) {
	codeArray := strings.Split(codeString, ",")

	/*	decodes := make (map[string]OptionDecode) */

	type optionDecode struct {
		Decode, OptionCode string
	}

	for _, code := range codeArray {

		for _, od := range decoder {
			if od.OptionCode == code {
				/*				decodes[od.OptionClass] = od */
				fmt.Printf("%s %s\n", od.OptionCode, od.Decode)
			}
		}
	}

	/*	fmt.Printf("%+v\n", decodes) */

}

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
			var model = "Unknown"
			var optionCodes = (*vehicles)[i].OptionCodes
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
	var idFound int
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

	/*
		chs, err := gotesla.GetChargeState(client, token, idFound)
		if err != nil {
			fmt.Printf("GetChargeState: %s\n", err)
			return
		}
		fmt.Printf("charge_state: %+v\n", chs)

		cls, err := gotesla.GetClimateState(client, token, idFound)
		if err != nil {
			fmt.Printf("GetClimateState: %s\n", err)
			return
		}
		fmt.Printf("climate_state: %+v\n", cls)

		ds, err := gotesla.GetDriveState(client, token, idFound)
		if err != nil {
			fmt.Printf("GetDriveState: %s\n", err)
			return
		}
		fmt.Printf("drive_state: %+v\n", ds)

		gs, err := gotesla.GetGuiSettings(client, token, idFound)
		if err != nil {
			fmt.Printf("GetGuiSettings: %s\n", err)
			return
		}
		fmt.Printf("gui_settings: %+v\n", gs)

		vs, err := gotesla.GetVehicleState(client, token, idFound)
		if err != nil {
			fmt.Printf("GetVehicleState: %s\n", err)
			return
		}
		fmt.Printf("vehicle_state: %+v\n", vs)

		vc, err := gotesla.GetVehicleConfig(client, token, idFound)
		if err != nil {
			fmt.Printf("GetVehicleConfig: %s\n", err)
			return
		}
		fmt.Printf("vehicle_config: %+v\n", vc)
	*/
	mobileEnabled, err := gotesla.GetMobileEnabled(client, token, idFound)
	if err != nil {
		fmt.Printf("GetMobileEnabled: %s\n", err)
		return
	}
	if *verbose {
		fmt.Printf("mobile_enabled: %+v\n", mobileEnabled)
	}

	vehicleData, err := gotesla.GetVehicleData(client, token, idFound)
	if err != nil {
		fmt.Printf("GetVehicleData: %s\n", err)
		return
	}
	if *verbose {
		fmt.Printf("vehicle_data: %+v\n", vehicleData)
	}

	// Print car ID stuff
	fmt.Printf("VIN: %s\n", vehicleData.Vehicle.Vin)
	fmt.Printf("ID: %d\n", vehicleData.Vehicle.Id)
	fmt.Printf("VehicleID: %d\n", vehicleData.Vehicle.VehicleId)
	fmt.Printf("DisplayName: %s\n", vehicleData.Vehicle.DisplayName)

	// Decode options
	fmt.Printf("OptionCodes: %s\n", vehicleData.Vehicle.OptionCodes)
	printOptionCodes(vehicleData.Vehicle.OptionCodes)

	return
}
