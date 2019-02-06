//
// Copyright (C) 2019 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//

//
// Client API package for Tesla vehicles
//
// This package wraps some (but by no means all) of the various
// API calls and data structures in the Tesla API.  Note that the
// API is not officially documented or supported; what is publically
// known has been reverse-engineered and collected at:
//
// https://www.tesla-api.timdorr.com/
//
// No attempt is made to document the functionality of the different
// API calls or data structures; for those details, please refer to the
// above Web site.
//
package gotesla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

// Tesla API parameters
var BaseUrl = "https://owner-api.teslamotors.com"
var UserAgent = "org.kitchenlab.gotesla"
var teslaClientId = "e4a9949fcfa04068f59abb5a658f2bac0a3428e4652315490b659d5ab3f35a9e"
var teslaClientSecret = "c75f14bbadc8bee3a7594412c31416f8300256d7668ea7e6e7f06727bfb9d220"

//
// Authentication
//
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	CreatedAt int `json:"created_at"`
}

//
// Authenticate with Tesla servers and get a bearer token.
//
func GetToken(client *http.Client, username *string, password *string) (Token, error) {
	var verbose bool = true
	
	// Auth is an authorization structure for the Tesla API.
	// Field names need to begin with capital letters for the JSON
	// package to marshall them, but we use field tags to make
	// the actual fields on the wire have the correct (all-lowercase)
	// capitalization.
	type Auth struct {
		GrantType string `json:"grant_type"`
		ClientId string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Email string `json:"email"`
		Password string `json:"password"`
	}
	var t Token
	
	// Figure out the correct endpoint
	var url = BaseUrl + "/oauth/token"
	if verbose {
		fmt.Printf("Login URL: %s\n", url)
	}
	
	// Create JSON structure for authentication request
	var auth Auth
	auth.GrantType = "password"
	auth.ClientId = teslaClientId
	auth.ClientSecret = teslaClientSecret
	auth.Email = *username
	auth.Password = *password
	
	authjson, err := json.Marshal(auth)
	if err != nil {
		return t, err
	}
	if verbose {
		fmt.Printf("Auth JSON: %s\n", authjson)
	}

	// Set up POST
  	req, err := http.NewRequest("POST", url, bytes.NewReader(authjson))
	if err != nil {
		return t, err
	}
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if verbose {
		fmt.Printf("Headers: %s\n", req.Header)
	}

	resp, err := client.Do(req) 
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return t, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &t)
	if err != nil {
		return t, nil
	}
	
	return t, nil
}

//
// General Tesla API request
//
func GetTesla(client *http.Client, bearerToken string, endpoint string) ([]byte, error) {
	var verbose bool = false
	
	// Figure out the correct endpoint
	var url = BaseUrl + endpoint
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
	req.Header.Add("Authorization", "Bearer " + bearerToken)

	if verbose {
		fmt.Printf("Headers: %s\n", req.Header)
	}

	resp, err := client.Do(req) 
	if err != nil {
		return nil, err
	}
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

//
// Vehicle Information queries
//
type Vehicle struct {
	Id int `json:"id"`
	VehicleId int `json:"vehicle_id"`
	Vin string `json:"vin"`
	DisplayName string `json:"display_name"`
	OptionCodes string `json:"option_codes"`
	Color interface{} `json:"color"`
	Tokens []string `json:"tokens"`
	State string `json:"state"`
	InService bool `json:"in_service"`
	IdS string `json:"id_s"`
	CalendarEnabled bool `json:'calendar_enabled'`
	ApiVersion int `json:'api_version'`
	BackseatToken interface{} `json:"backseat_token"`
	BackseatTokenUpdatedAt interface{} `json:"backseat_token_updated_at"`
}

type Vehicles []struct {
	*Vehicle
}

type VehiclesResponse struct {
	Response Vehicles `json:"response"`
	Count int `json:"count"`
}

func GetVehicles(client *http.Client, bearerToken string) (VehiclesResponse, error) {
	var verbose = false
	var vr VehiclesResponse

	vehiclejson, err := GetTesla(client, bearerToken, "/api/1/vehicles")
	if err != nil {
		return vr, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}
	
	err = json.Unmarshal(vehiclejson, &vr)
	if err != nil {
		return vr, err
	}

	return vr, nil
}

// Nearby Charging Sites

type ChargerLocation struct {
	Lat float64 `json:"lat"`
	Long float64 `json:"long"`
}

type Charger struct {
	Location ChargerLocation `json:"location"`
	Name string `json:"name"`
	Type string `json:"type"` // "destination" or "supercharger"
	DistanceMiles float64 `json:"distance_miles"`
}

type DestinationCharger struct {
	Charger
}

type Supercharger struct {
	Charger
	AvailableStalls int `json:"available_stalls"`
	TotalStalls int `json:"total_stalls"`
	SiteClosed bool `json:"site_closed"`
}

type NearbyChargingSitesResponse struct {
	Response struct {
		CongestionSyncTimeUtcSecs int `json:"congestion_sync_time_utc_secs"`
		DestinationCharging []DestinationCharger `json:"destination_charging"`
		Superchargers []Supercharger `json:"superchargers"`
		Timestamp int `json:"timestamp"`
	}
}

func GetNearbyChargers(client *http.Client, bearerToken string, id int) (NearbyChargingSitesResponse, error) {
	var verbose = false
	var ncsr NearbyChargingSitesResponse

	vehiclejson, err := GetTesla(client, bearerToken, "/api/1/vehicles/" + strconv.Itoa(id) + "/nearby_charging_sites")
	if err != nil {
		return ncsr, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}
	
	err = json.Unmarshal(vehiclejson, &ncsr)
	if err != nil {
		return ncsr, err
	}

	return ncsr, nil
}

