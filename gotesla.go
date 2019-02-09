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
// https://www.teslaapi.io/
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
	"os"
	"strconv"
	"time"
)

// Tesla API parameters
var BaseUrl = "https://owner-api.teslamotors.com"
var UserAgent = "org.kitchenlab.gotesla"
var teslaClientId = "e4a9949fcfa04068f59abb5a658f2bac0a3428e4652315490b659d5ab3f35a9e"
var teslaClientSecret = "c75f14bbadc8bee3a7594412c31416f8300256d7668ea7e6e7f06727bfb9d220"

// Place to cache token credentials
// This is pretty UNIX-specific
var TokenCachePath = os.Getenv("HOME") + "/.gotesla.cache"
var TokenCachePathNewSuffix = ".new"

//
// Authentication
//

// Auth is an authorization structure for the Tesla API.
// Field names need to begin with capital letters for the JSON
// package to marshall them, but we use field tags to make
// the actual fields on the wire have the correct (all-lowercase)
// capitalization.
//
// A user can either authenticate with an email and password,
// or if re-authenticating (refreshing a token), pass the
// refresh token.
//
type Auth struct {
	GrantType string `json:"grant_type"`
	ClientId string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Email string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Token is basically an OAUTH 2.0 bearer token plus some metadata.
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
func GetToken(client *http.Client, username *string, password *string) (*Token, error) {

	// Create JSON structure for authentication request
	var auth Auth
	auth.GrantType = "password"
	auth.ClientId = teslaClientId
	auth.ClientSecret = teslaClientSecret
	auth.Email = *username
	auth.Password = *password
	
	// call common code
	return tokenAuthCommon(client, &auth)
}

//
// Refresh an existing token
//
func RefreshToken(client *http.Client, token *Token) (*Token, error) {

	// Create JSON structure for authentication request
	var auth Auth
	auth.GrantType = "refresh_token"
	auth.ClientId = teslaClientId
	auth.ClientSecret = teslaClientSecret
	auth.RefreshToken = token.RefreshToken

	// call common code
	return tokenAuthCommon(client, &auth)
}

// Common authentication code used by GetToken and RefreshToken.
// Basically passes an authentication structure to Telsa and
// gets back a Token.
func tokenAuthCommon(client *http.Client, auth *Auth) (*Token, error) {
	var verbose bool = false
	var t Token

	authjson, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Auth JSON: %s\n", authjson)
	}

	body, err := PostTesla(client, nil, "/oauth/token", authjson)
	
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}
	
	return &t, nil
}

//
// Save token
// Write the token to the new file and if that succeeds, move it
// atomically into place
//
func SaveCachedToken(t *Token) error {

	// Convert the token structure to JSON
	tokenJSON, err := json.Marshal(t)
	if err != nil {
		return err
	}
	
	// Write it to the file
	err = ioutil.WriteFile(TokenCachePath + TokenCachePathNewSuffix, tokenJSON, 0600)
	if err != nil {
		return err
	}
	
	// Move into place
	err = os.Rename(TokenCachePath + TokenCachePathNewSuffix, TokenCachePath)
	if err != nil {
		return err
	}

	return nil
}

// Get a token and cache it in the local filesystem
// This function is preferred over GetToken because it (in theory anyway)
// should result in fewer authentication calls to Tesla's servers due to
// caching.
func GetAndCacheToken(client *http.Client, username *string, password *string) (*Token, error) {
	t, err := GetToken(client, username, password)
	if err != nil {
		return t, err
	}
	err = SaveCachedToken(t)
	if err != nil {
		return t, err
	}

	return t, nil
}

// Refresh a token and cache it in the local filesystem
// This function is preferred over RefreshToken.
func RefreshAndCacheToken(client *http.Client, token *Token) (*Token, error) {
	t, err := RefreshToken(client, token)
	if err != nil {
		return t, err
	}
	err = SaveCachedToken(t)
	if err != nil {
		return t, err
	}

	return t, nil
}

// Load the token from the cache file
func LoadCachedToken() (*Token, error) {
	var t Token
	
	body, err := ioutil.ReadFile(TokenCachePath)
	if err != nil {
		return nil, err
	}

	// Parse response, get token structure
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}
	
	return &t, nil
}

// Delete cached token
func DeleteCachedToken() (error) {
	err := os.Remove(TokenCachePath)
	return err
}

// Return true if a token is valid
func CheckToken(t *Token) (bool) {
	start, end := TokenTimes(t)
	now := time.Now()
	return (start.Before(now) && now.Before(end))
}

// Retrieve start and end times for a token
func TokenTimes(t *Token) (start, end time.Time) {
	start = time.Unix(int64(t.CreatedAt), 0)
	end = time.Unix(int64(t.CreatedAt) + int64(t.ExpiresIn), 0)
	return
}

//
// General Tesla API requests
//
func GetTesla(client *http.Client, token *Token, endpoint string) ([]byte, error) {
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
	if token != nil {
		req.Header.Add("Authorization", "Bearer " + token.AccessToken)
	}

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

func PostTesla(client *http.Client, token *Token, endpoint string, payload []byte) ([]byte, error) {
	var verbose bool = false

	// Compute endpoint URL
	var url = BaseUrl + endpoint
	if verbose {
		fmt.Printf("URL: %s\n", url)
	}

	// Set up POST
  	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if token != nil {
		req.Header.Add("Authorization", "Bearer " + token.AccessToken)
	}

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

func GetVehicles(client *http.Client, token *Token) (VehiclesResponse, error) {
	var verbose = false
	var vr VehiclesResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles")
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

func GetNearbyChargers(client *http.Client, token *Token, id int) (NearbyChargingSitesResponse, error) {
	var verbose = false
	var ncsr NearbyChargingSitesResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/" + strconv.Itoa(id) + "/nearby_charging_sites")
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

