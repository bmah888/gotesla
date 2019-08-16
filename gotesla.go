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

// BaseUrl is the leading part of the API URL.  It is unlikely to ever change.
var BaseUrl = "https://owner-api.teslamotors.com"

// UserAgent is passed in HTTP requests to the Tesla API.
// This appears to be a mandatory parameter; the API will not work without
// some value being passed here.
var UserAgent = "org.kitchenlab.gotesla"
var teslaClientId = "e4a9949fcfa04068f59abb5a658f2bac0a3428e4652315490b659d5ab3f35a9e"
var teslaClientSecret = "c75f14bbadc8bee3a7594412c31416f8300256d7668ea7e6e7f06727bfb9d220"

// TokenCachePath is the location (UNIX specific?) to cache API credentials.
var TokenCachePath = os.Getenv("HOME") + "/.gotesla.cache"

// TokenCachePathNewSuffix is the suffix to add to a new cache file when updating.
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
	var verbose = false
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
	var verbose = false

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
	var verbose = false

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
	CalendarEnabled        bool        `json:"calendar_enabled"`
	ApiVersion             int         `json:"api_version"`
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
func GetVehicles(client *http.Client, token *Token) (*Vehicles, error) {
	var verbose = false
	var vr VehiclesResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &vr)
	if err != nil {
		return nil, err
	}

	// Check Count to see if it matches the length of Response
	if len(vr.Response) != vr.Count {
		return nil, fmt.Errorf("get_vehicles Response length %d != Count %d", len(vr.Response), vr.Count)
	}

	return &(vr.Response), nil
}

// ChargeStateResponse is the return from a charge_state call
type ChargeStateResponse struct {
	Response ChargeState
}

// ChargeState is the actual charge_state data
type ChargeState struct {
	BatteryHeaterOn              bool        `json:"battery_heater_on"`
	BatteryLevel                 int         `json:"battery_level"`
	BatteryRange                 float64     `json:"battery_range"`
	ChargeCurrentRequest         int         `json:"charge_current_request"`
	ChargeCurrentRequestMax      int         `json:"charge_current_request_max"`
	ChargeEnableRequest          bool        `json:"charge_enable_request"`
	ChargeLimitSoc               int         `json:"charge_limit_soc"`
	ChargeLimitSocMax            int         `json:"charge_limit_soc_max"`
	ChargeLimitSocMin            int         `json:"charge_limit_soc_min"`
	ChargeLimitSocStd            int         `json:"charge_limit_soc_std"`
	ChargeMilesAddedIdeal        float64     `json:"charge_miles_added_ideal"`
	ChargeMilesAddedRated        float64     `json:"charge_miles_added_rated"`
	ChargePortColdWeatherMode    bool        `json:"charge_port_cold_weather_mode"`
	ChargePortDoorOpen           bool        `json:"charge_port_door_open"`
	ChargePortLatch              string      `json:"charge_port_latch"` // "Engaged", "Disengaged"
	ChargeRate                   float64     `json:"charge_rate"`
	ChargeToMaxRange             bool        `json:"charge_to_max_range"`
	ChargerActualCurrent         int         `json:"charge_actual_current"`
	ChargerPhases                int         `json:"charge_phases"` // 1?
	ChargerPilotCurrent          int         `json:"charger_pilot_current"`
	ChargerPower                 int         `json:"charger_power"`
	ChargerVoltage               int         `json:"charger_voltage"`
	ChargingState                string      `json:"charging_state"` // "Stopped", "Starting", "Charging", "Disconnected"
	ConnChargeCable              string      `json:"conn_charge_cable"`
	EstBatteryRange              float64     `json:"est_battery_range"`
	FastChargerBrand             string      `json:"fast_charger_brand"`
	FastChargerPresent           bool        `json:"fast_charger_present"`
	FastChargerType              string      `json:"fast_charger_type"`
	IdealBatteryRange            float64     `json:"ideal_battery_range"`
	ManagedChargingActive        bool        `json:"managed_charging_active"`
	ManagedChargingStartTime     interface{} `json:"managed_charging_start_time"`
	ManagedChargingUserCancelled bool        `json:"managed_charging_user_cancelled"`
	MaxRangeChargeCounter        int         `json:"max_range_charge_counter"`
	NotEnoughPowerToHeat         bool        `json:"not_enough_power_to_heat"`
	ScheduledChargingPending     bool        `json:"scheduled_charging_pending"`
	ScheduledChargingStartTime   int         `json:"scheduled_charging_start_time"` // seconds
	TimeToFullCharge             float64     `json:"time_to_full_charge"`           // in hours
	TimeStamp                    int         `json:"timestamp"`                     // ms
	TripCharging                 bool        `json:"trip_charging"`
	UsableBatteryLevel           int         `json:"usable_battery_level"`
	UserChargeEnableRequest      bool        `json:"user_charge_enable_request"`
}

// GetChargeState retrieves the state of charge in the battery and various settings
func GetChargeState(client *http.Client, token *Token, id int) (*ChargeState, error) {
	var verbose = false
	var csr ChargeStateResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/charge_state")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &csr)
	if err != nil {
		return nil, err
	}
	return &(csr.Response), nil
}

// ClimateStateResponse encapsulates a ClimateState object
type ClimateStateResponse struct {
	Response ClimateState
}

// ClimateState returns the state of the climate control
type ClimateState struct {
	BatteryHeater              bool    `json:"battery_heater"`
	BatteryHeaterNoPower       bool    `json:"battery_heater_no_power"`
	DriverTempSetting          float64 `json:"driver_temp_setting"`
	FanStatus                  int     `json:"fan_status"`
	InsideTemp                 float64 `json:"inside_temp"`
	IsAutoConditioningOn       bool    `json:"is_auto_conditioning_on"`
	IsClimateOn                bool    `json:"is_climate_on"`
	IsFrontDefrosterOn         bool    `json:"is_front_defroster_on"`
	IsPreconditioning          bool    `json:"is_preconditioning"`
	IsRearDefrosterOn          bool    `json:"is_rear_defroster_on"`
	LeftTempDirection          int     `json:"left_temp_direction"`
	MaxAvailTemp               float64 `json:"max_avail_temp"`
	MinAvailTemp               float64 `json:"min_avail_temp"`
	OutsideTemp                float64 `json:"outside_temp"`
	PassengerTempSetting       float64 `json:"passenger_temp_setting"`
	RemoteHeaterControlEnabled bool    `json:"remote_heater_control_enabled"`
	RightTempDirection         int     `json:"right_temp_direction"`
	SeatHeaterLeft             int     `json:"seat_heater_left"`
	SeatHeaterRearCenter       int     `json:"seat_heater_rear_center"`
	SeatHeaterRearLeft         int     `json:"seat_heater_rear_left"`
	SeatHeaterRearLeftBack     int     `json:"seat_heater_rear_left_back"`
	SeatHeaterRearRight        int     `json:"seat_heater_rear_right"`
	SeatHeaterRearRightBack    int     `json:"seat_heater_rear_right_back"`
	SeatHeaterRight            int     `json:"seat_heater_right"`
	SideMirrorHeaters          bool    `json:"side_mirror_heaters"`
	SmartPreconditioning       bool    `json:"smart_preconditioning"`
	SteeringWheelHeater        bool    `json:"steering_wheel_heater"`
	TimeStamp                  int     `json:"timestamp"` // ms
	WiperBladeHeater           bool    `json:"wiper_blade_heater"`
}

// GetClimateState returns information on the current internal
// temperature and climate control system.
func GetClimateState(client *http.Client, token *Token, id int) (*ClimateState, error) {
	var verbose = false
	var clsr ClimateStateResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/climate_state")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &clsr)
	if err != nil {
		return nil, err
	}

	return &(clsr.Response), nil
}

// DriveStateResponse encapsulates a DriveState object.
type DriveStateResponse struct {
	Response DriveState
}

// DriveState is the result of the drive_state call, and includes information
// about vehicle position and speed
type DriveState struct {
	GpsAsOf                 int         `json:"gps_as_of"`
	Heading                 int         `json:"heading"`
	Latitude                float64     `json:"latitude"`
	Longitude               float64     `json:"longitude"`
	NativeLatitude          float64     `json:"native_latitude"`
	NativeLocationSupported int         `json:"native_location_supported"`
	NativeLongitude         float64     `json:"native_longitude"`
	NativeType              string      `json:"native_type"`
	Power                   int         `json:"power"`
	ShiftState              interface{} `json:"shift_state"`
	Speed                   interface{} `json:"speed"`
	TimeStamp               int         `json:"timestamp"` // ms
}

// GetDriveState returns the driving and position state of the vehicle
func GetDriveState(client *http.Client, token *Token, id int) (*DriveState, error) {
	var verbose = false
	var dsr DriveStateResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/drive_state")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &dsr)
	if err != nil {
		return nil, err
	}

	return &(dsr.Response), nil
}

// GuiSettingsResponse encapsulates a GuiSettings object
type GuiSettingsResponse struct {
	Response GuiSettings
}

// GuiSettings return a number of settings regarding the GUI on the CID
type GuiSettings struct {
	Gui24HourTime       bool   `json:"gui_24_hour_time"`
	GuiChargeRateUnits  string `json:"gui_charge_rate_units"`
	GuiDistanceUnits    string `json:"gui_distance_units"`
	GuiRangeDisplay     string `json:"gui_range_display"`
	GuiTemperatureUnits string `json:"gui_temperature_units"`
	TimeStamp           int    `json:"timestamp"` // ms
}

// GetGuiSettings returns various information about the GUI settings
// of the car, such as unit format and range display
func GetGuiSettings(client *http.Client, token *Token, id int) (*GuiSettings, error) {
	var verbose = false
	var gsr GuiSettingsResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/gui_settings")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &gsr)
	if err != nil {
		return nil, err
	}
	return &(gsr.Response), nil
}

// VehicleStateResponse encapsulates a VehicleState object
type VehicleStateResponse struct {
	Response VehicleState
}

// VehicleState is the return value from a vehicle_state call
type VehicleState struct {
	ApiVersion              int                        `json:"api_version"`
	AutoparkStateV2         string                     `json:"autopark_state_v2"`
	AutoparkStyle           string                     `json:"autopark_style"`
	CalendarSupported       bool                       `json:"calendar_supported"`
	CarVersion              string                     `json:"car_version"`
	CenterDisplayState      int                        `json:"center_display_state"`
	Df                      int                        `json:"df"`
	Dr                      int                        `json:"dr"`
	Ft                      int                        `json:"ft"`
	HomelinkNearby          bool                       `json:"homelink_nearby"`
	IsUserPresent           bool                       `json:"is_user_present"`
	LastAutoparkError       string                     `json:"last_autopark_error"`
	Locked                  bool                       `json:"locked"`
	MediaState              VehicleStateMediaState     `json:"media_state"`
	NotificationsSupported  bool                       `json:"notifications_supported"`
	Odometer                float64                    `json:"odometer"`
	ParsedCalendarSupported bool                       `json:"parsed_calendar_supported"`
	Pf                      int                        `json:"pf"`
	Pr                      int                        `json:"pr"`
	RemoteStart             bool                       `json:"remote_start"`
	RemoteStartSupported    bool                       `json:"remote_start_started"`
	Rt                      int                        `json:"rt"`
	SoftwareUpdate          VehicleStateSoftwareUpdate `json:"software_update"`
	SpeedLimitMode          VehicleStateSpeedLimitMode `json:"speed_limit_mode"`
	SunRoofPercentOpen      int                        `json:"sun_roof_percent_open"`
	SunRoofState            string                     `json:"sun_roof_state"`
	TimeStamp               int                        `json:"timestamp"` // ms
	ValetMode               bool                       `json:"valet_mode"`
	ValetPinNeeded          bool                       `json:"valet_pin_needed"`
	VehicleName             string                     `json:"vehicle_name"`
}

// A VehicleStateMediaState returns the state of media control
type VehicleStateMediaState struct {
	RemoteControlEnabled bool `json:"remote_control_enabled"`
}

// A VehicleStateSoftwareUpdate returns information on pending software updates
type VehicleStateSoftwareUpdate struct {
	ExpectedDurationSec int    `json:"expected_duration_sec"`
	Status              string `json:"status"`
}

// A VehicleStateSpeedLimitMode returns the speed limiting parameters
type VehicleStateSpeedLimitMode struct {
	Active          bool    `json:"active"`
	CurrentLimitMph float64 `json:"current_limit_mph"`
	MaxLimitMph     int     `json:"max_limit_mph"`
	MinLimitMph     int     `json:"min_limit_mph"`
	PinCodeSet      bool    `json:"pin_code_set"`
}

// GetVehicleState returns the vehicle's physical state, such as which
// doors are open.
func GetVehicleState(client *http.Client, token *Token, id int) (*VehicleState, error) {
	var verbose = false
	var vsr VehicleStateResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/vehicle_state")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &vsr)
	if err != nil {
		return nil, err
	}
	return &(vsr.Response), nil
}

// VehicleConfigResponse encapsulates a VehicleConfig
type VehicleConfigResponse struct {
	Response VehicleConfig
}

// VehicleConfig is the return data from a vehicle_config call
type VehicleConfig struct {
	CanAcceptNavigationRequests bool   `json:"can_accept_navigation_requests"`
	CanActuateTrunks            bool   `json:"can_actuate_trunks"`
	CarSpecialType              string `json:"car_special_type"` // "base"
	CarType                     string `json:"car_type"`         // "models"
	ChargePortType              string `json:"charge_port_type"`
	EuVehicle                   bool   `json:"eu_vehicle"`
	ExteriorColor               string `json:"exterior_color"`
	HasAirSuspension            bool   `json:"has_air_suspension"`
	HasLudicrousMode            bool   `json:"has_ludicrous_mode"`
	MotorizedChargePort         bool   `json:"motorized_charge_port"`
	PerfConfig                  string `json:"perf_config"`
	Plg                         bool   `json:"plg"`
	RearSeatHeaters             int    `json:"rear_seat_heaters"`
	RearSeatType                int    `json:"rear_seat_type"`
	Rhd                         bool   `json:"rhd"`
	RoofColor                   string `json:"roof_color"` // "Colored"
	SeatType                    int    `json:"seat_type"`
	SpoilerType                 string `json:"spoiler_type"`
	SunRoofInstalled            int    `json:"sun_roof_installed"`
	ThirdRowSeats               string `json:"third_row_seats"`
	TimeStamp                   int    `json:"timestamp"` // ms
	TrimBadging                 string `json:"trim_badging"`
	WheelType                   string `json:"wheel_type"`
}

// GetVehicleConfig performs a vehicle_config call
func GetVehicleConfig(client *http.Client, token *Token, id int) (*VehicleConfig, error) {
	var verbose = false
	var vcr VehicleConfigResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/data_request/vehicle_config")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &vcr)
	if err != nil {
		return nil, err
	}
	return &(vcr.Response), nil
}

// VehicleDataResponse is the return from a vehicle_data call
type VehicleDataResponse struct {
	Response VehicleData
}

// VehicleData is the actual data structure for a vehicle_data call
type VehicleData struct {
	Vehicle
	UserId int           `json:"user_id"`
	Ds     DriveState    `json:"drive_state"`
	Cls    ClimateState  `json:"climate_state"`
	Chs    ChargeState   `json:"charge_state"`
	Gs     GuiSettings   `json:"gui_settings"`
	Vs     VehicleState  `json:"vehicle_state"`
	Vc     VehicleConfig `json:"vehicle_config"`
}

// GetVehicleData performs a vehicle_data call
func GetVehicleData(client *http.Client, token *Token, id int) (*VehicleData, error) {
	var verbose = false
	var vdr VehicleDataResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/vehicle_data")
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &vdr)
	if err != nil {
		return nil, err
	}
	return &(vdr.Response), nil
}

// MobileEnabledResponse is the return from a mobile_enabled call
type MobileEnabledResponse struct {
	Response bool `json:"response"`
}

// GetMobileEnabled returns whether mobile access is enabled
func GetMobileEnabled(client *http.Client, token *Token, id int) (bool, error) {
	var verbose = false
	var mer MobileEnabledResponse

	vehiclejson, err := GetTesla(client, token, "/api/1/vehicles/"+strconv.Itoa(id)+"/mobile_enabled")
	if err != nil {
		return false, err
	}
	if verbose {
		fmt.Printf("Response: %s\n", vehiclejson)
	}

	err = json.Unmarshal(vehiclejson, &mer)
	if err != nil {
		return false, err
	}

	return mer.Response, nil
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
