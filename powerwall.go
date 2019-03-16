//
// Copyright (C) 2019 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//

package gotesla

import (
//	"bytes"
	"encoding/json"
//	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
//	"os"
//	"strconv"
//	"time"
)

// Tesla API parameters

type Meter struct {
	LastCommunicationTime string  `json:"last_communication_time"`
	InstantPower          float64 `json:"instant_power"`
	InstantReactivePower  float64 `json:"instant_reactive_power"`
	InstantApparentPower  float64 `json:"instant_apparent_power"`
	Frequency             float64 `json:"frequency"`
	EnergyExported        float64 `json:"energy_exported"`
	EnergyImported        float64 `json:"energy_imported"`
	InstantAverageVoltage float64 `json:"instant_average_voltage"`
	InstantTotalCurrent   float64 `json:"instant_total_current"`
	InstantACurrent       float64 `json:"instant_a_current"`
	InstantBCurrent       float64 `json:"instant_b_current"`
	InstantCCurrent       float64 `json:"instant_c_current"`
	Timeout               int     `json:"timeout"`
}

type MeterAggregate struct {
	Site      Meter
	Battery   Meter
	Load      Meter
	Solar     Meter
	Busway    Meter
	Frequency Meter
	Generator Meter
}

func GetMeterAggregate(client *http.Client, hostname string) (*MeterAggregate, error) {
	var verbose bool = false
	var ma MeterAggregate

	body, err := GetPowerwall(client, hostname, "/api/meters/aggregates")

	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &ma)
	if err != nil {
		return nil, err
	}

	return &ma, nil
}

type Soe struct {
	Percentage float64 `json:"percentage"`
}

func GetSoe(client *http.Client, hostname string) (float64, error) {
	var verbose bool = false
	var soe Soe

	body, err := GetPowerwall(client, hostname, "/api/system_status/soe")

	if err != nil {
		return 0.0, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &soe)
	if err != nil {
		return 0.0, err
	}

	return soe.Percentage, nil
}

type GridStatusResponse struct {
	GridStatus string `json:"grid_status"`
}
const gridStatusUpString string = "SystemGridConnected"
const gridStatusDownString string = "SystemIslandedActive"
const gridStatusTransitionString string = "SystemTransitionToGrid"
type GridStatus int
const (
	GridStatusUnknown GridStatus = iota
	GridStatusDown
	GridStatusTransition
	GridStatusUp
)

func GetGridStatus(client *http.Client, hostname string) (GridStatus, error) {
	var verbose bool = false
	var gsr GridStatusResponse

	body, err := GetPowerwall(client, hostname, "/api/system_status/grid_status")

	if err != nil {
		return GridStatusUnknown, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &gsr)
	if err != nil {
		return GridStatusUnknown, err
	}

	var gs GridStatus = GridStatusUnknown
	switch gsr.GridStatus {
	case gridStatusUpString:
		gs = GridStatusUp
	case gridStatusDownString:
		gs = GridStatusDown
	case gridStatusTransitionString:
		gs = GridStatusTransition
	}

	return gs, nil
}

// GetPowerwall performs a GET request to a local Tesla Powerwall gateway.
// It doesn't do authentication yet.
func GetPowerwall(client *http.Client, hostname string, endpoint string) ([]byte, error) {

	var verbose bool = false

	// Figure out the correct endpoint
	var url = "https://" + hostname + endpoint
	if verbose {
		fmt.Printf("URL: %s\n", url)
	}

	// Set up GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	//	if token != nil {
	//		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	//	}

	if verbose {
		fmt.Printf("Headers: %s\n", req.Header)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Try to handle certain types of HTTP status codes
	if verbose {
		fmt.Printf("Status: %s\n", resp.Status)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		/* break */
	default:
		return nil, fmt.Errorf("%s", http.StatusText(resp.StatusCode))
	}

	// If we get here, we can be reasonably (?) assured of a valid body.
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Caller needs to parse this in the context of whatever schema it knows
	return body, nil

}
