//
// Copyright (C) 2019 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//

//
// Package gotesla is a client library for Tesla vehicles
//
// This package wraps some (but by no means all) of the various
// API calls and data structures in the Tesla API.  Note that the
// API is not officially documented or supported; what is publically
// known has been reverse-engineered and collected at:
//
// https://tesla-api.timdorr.com/
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
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Token is basically an OAUTH 2.0 bearer token plus some metadata.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	CreatedAt    int    `json:"created_at"`
}

//
// GetToken authenticates with Tesla servers and returns a Token
// structure.
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
// RefreshToken refreshes an existing token and returns a new Token
// structure.
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
// SaveCachedToken saves a Token structure (JSON representation)
// in a file that is by default in the user's home directory.
// Writes the token to a temporary file and if that succeeds, move it
// atomically into place.
//
func SaveCachedToken(t *Token) error {

	// Convert the token structure to JSON
	tokenJSON, err := json.Marshal(t)
	if err != nil {
		return err
	}

	// Write it to the file
	err = ioutil.WriteFile(TokenCachePath+TokenCachePathNewSuffix, tokenJSON, 0600)
	if err != nil {
		return err
	}

	// Move into place
	err = os.Rename(TokenCachePath+TokenCachePathNewSuffix, TokenCachePath)
	if err != nil {
		return err
	}

	return nil
}

// GetAndCacheToken gets a new token and saves it in the local filesystem.
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

// RefreshAndCacheToken does a refresh and saves the returned token in
// the local filesystem
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

// LoadCachedToken returns the token (if any) from the cache file.
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

// DeleteCachedToken removes the cached token file.
func DeleteCachedToken() error {
	err := os.Remove(TokenCachePath)
	return err
}

// CheckToken returns true if a token is valid.
func CheckToken(t *Token) bool {

	// Currently the only check is for timestamp validity, which
	// assume the local clock is synchronized.
	start, end := TokenTimes(t)
	now := time.Now()
	return (start.Before(now) && now.Before(end))
}

// TokenTimes returns the start and end times for a token.
func TokenTimes(t *Token) (start, end time.Time) {
	start = time.Unix(int64(t.CreatedAt), 0)
	end = time.Unix(int64(t.CreatedAt)+int64(t.ExpiresIn), 0)
	return
}

//
// General Tesla API requests
//

// GetTesla performs a GET request to the Tesla API.
// If a non-nil authentication Token structure is passed, the bearer
// token part is used to authenticate the request.
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
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	}

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

// PostTesla performs an HTTP POST request to the Tesla API.
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
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
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

// Vehicle is a structure that describes a single Tesla vehicle.
type Vehicle struct {
	Id                     int         `json:"id"`
	VehicleId              int         `json:"vehicle_id"`
	Vin                    string      `json:"vin"`
	DisplayName            string      `json:"display_name"`
	OptionCodes            string      `json:"option_codes"`
	Color                  interface{} `json:"color"`
	Tokens                 []string    `json:"tokens"`
	State                  string      `json:"state"`
	InService              bool        `json:"in_service"`
	IdS                    string      `json:"id_s"`
	CalendarEnabled        bool        `json:'calendar_enabled'`
	ApiVersion             int         `json:'api_version'`
	BackseatToken          interface{} `json:"backseat_token"`
	BackseatTokenUpdatedAt interface{} `json:"backseat_token_updated_at"`
}

// Vehicles encapsulates a collection of Tesla Vehicles.
type Vehicles []struct {
	*Vehicle
}

// VehiclesResponse is the response to a vehicles API query.
type VehiclesResponse struct {
	Response Vehicles `json:"response"`
	Count    int      `json:"count"`
}

// GetVehicles performs a vehicles query to retrieve information on all
// the Tesla vehicles associated with an account.
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

// ChargerLocation represents the physical coordinates of a charging station.
type ChargerLocation struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

// Charger represents information common to all Tesla chargers.
type Charger struct {
	Location      ChargerLocation `json:"location"`
	Name          string          `json:"name"`
	Type          string          `json:"type"` // "destination" or "supercharger"
	DistanceMiles float64         `json:"distance_miles"`
}

// DestinationCharger represents a Tesla Destination charger.
type DestinationCharger struct {
	Charger
}

// Supercharger represents a Tesla Supercharger.
// In addition to the common Charger fields, this also includes
// information on stall occupancy.
type Supercharger struct {
	Charger
	AvailableStalls int  `json:"available_stalls"`
	TotalStalls     int  `json:"total_stalls"`
	SiteClosed      bool `json:"site_closed"`
}

// NearbyChargingSitesResponse encapsulates the response to a
// nearby_charging_sites API query on a given vehicle.  Note that
// queries are specific to a given vehicle.
type NearbyChargingSitesResponse struct {
	Response struct {
		CongestionSyncTimeUtcSecs int                  `json:"congestion_sync_time_utc_secs"`
		DestinationCharging       []DestinationCharger `json:"destination_charging"`
		Superchargers             []Supercharger       `json:"superchargers"`
		Timestamp                 int                  `json:"timestamp"`
	}
}

// GetNearbyChargers retrieves the chargers closest to a given vehicle.
func GetNearbyChargers(client *http.Client, token *Token, id int) (NearbyChargingSitesResponse, error) {
	var verbose = false
	var ncsr NearbyChargingSitesResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/nearby_charging_sites")
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
