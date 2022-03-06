//
// Copyright (C) 2019-2021 Bruce A. Mah.
// All rights reserved.
//
// Distributed under a BSD-style license, see the LICENSE file for
// more information.
//

package gotesla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	pb "github.com/bmah888/gotesla/teslapowerpb"
	"google.golang.org/protobuf/proto"
)

// Tesla API parameters

// Authentication
type PowerwallAuth struct {
	Email     string `json:"email"`
	Token     string `json:"token"`
	LoginTime string `json:"loginTime"`
	Timestamp time.Time
}

// A Meter contains the state of one of the (four?) energy meters
// attached to the gateway.
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

// A MeterAggregate contains several Meters.
// Four of them correspond
// to the four energy sources in the Tesla app (Site, Battery, Load,
// and Solar).  The remaining three are unknown at this time.
type MeterAggregate struct {
	Site      Meter
	Battery   Meter
	Load      Meter
	Solar     Meter
	Busway    Meter
	Frequency Meter
	Generator Meter
}

// GetMeterAggregate retrieves a MeterAggregate from a local
// Powerwall gateway.  No authentication is required for this
// call.
func GetMeterAggregate(client *http.Client, hostname string, pwa *PowerwallAuth) (*MeterAggregate, error) {
	var verbose = false
	var ma MeterAggregate

	body, err := GetPowerwall(client, hostname, "/api/meters/aggregates", pwa)

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

type SystemStatusResponse struct {
	BatteryTargetPower     float64        `json:"battery_target_power"`
	NominalFullPackEnergy  int            `json:"nominal_full_pack_energy"`
	NominalEnergyRemaining int            `json:"nominal_energy_remaining"`
	AvailableBlocks        int            `json:"available_blocks"`
	BatteryBlocks          []BatteryBlock `json:"battery_blocks"`
	SystemIslandState      string         `json:"system_island_state"`
}

type BatteryBlock struct {
	PackagePartNumber      string
	PackageSerialNumber    string
	NominalFullPackEnergy  int `json:"nominal_full_pack_energy"`
	NominalEnergyRemaining int `json:"nominal_energy_remaining"`
	EnergyCharged          int `json:"energy_charged"`
	EnergyDischarged       int `json:"energy_discharged"`
}

func GetSystemStatus(client *http.Client, hostname string, pwa *PowerwallAuth) (*SystemStatusResponse, error) {
	var verbose = false
	var sysstat SystemStatusResponse

	body, err := GetPowerwall(client, hostname, "/api/system_status", pwa)

	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	err = json.Unmarshal(body, &sysstat)
	if err != nil {
		return nil, err
	}

	return &sysstat, nil
}

// A Soe structure gives the current state of energy of the Powerwall
// batteries (total, as a value between 0-100).
type Soe struct {
	Percentage float64 `json:"percentage"`
}

// GetSoe returns the state of energy of the Powerwall batteries.
// Unlike some other calls in this library, it doesn't return the
// structure, just a float64 value (and error if applicable).
func GetSoe(client *http.Client, hostname string, pwa *PowerwallAuth) (float64, error) {
	var verbose = false
	var soe Soe

	body, err := GetPowerwall(client, hostname, "/api/system_status/soe", pwa)

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

// GridStatusResponse is a structure that gives the current grid
// status as a string, as defined in the following constants.
type GridStatusResponse struct {
	GridStatus string `json:"grid_status"`
}

const gridStatusUpString string = "SystemGridConnected"
const gridStatusDownString string = "SystemIslandedActive"
const gridStatusTransitionString string = "SystemTransitionToGrid"

// GridStatus tracks the status of the grid connection to the house
type GridStatus int

// GridStatus values
const (
	GridStatusUnknown GridStatus = iota
	GridStatusDown
	GridStatusTransition
	GridStatusUp
)

// GetGridStatus returns the grid status as a GridStatus value.
// We do it this way in order to avoid the caller needing to parse
// the response strings.
func GetGridStatus(client *http.Client, hostname string, pwa *PowerwallAuth) (GridStatus, error) {
	var verbose = false
	var gsr GridStatusResponse

	body, err := GetPowerwall(client, hostname, "/api/system_status/grid_status", pwa)

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

	var gs = GridStatusUnknown
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

// SiteMasterResponse
type SiteMasterResponse struct {
	Running          bool   `json:"running"`
	Uptime           string `json:"uptime"`
	ConnectedToTesla bool   `json:"connected_to_tesla"`
}

func GetSiteMaster(client *http.Client, hostname string, pwa *PowerwallAuth) (*SiteMasterResponse, error) {
	var verbose = false
	var smr SiteMasterResponse

	body, err := GetPowerwall(client, hostname, "/api/sitemaster", pwa)

	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Printf("Resp JSON %s\n", body)
	}

	// Parse response, get the sitemaster structure
	err = json.Unmarshal(body, &smr)
	if err != nil {
		return nil, err
	}

	return &smr, nil
}

type VitalDevices struct {
	STSTSM      STSTSM
	TESYNC      TESYNC
	TEMSA       TEMSA
	TETHCs      []TETHC
	TEPODs      []TEPOD
	TEPINVs     []TEPINV
	PVACs       []PVAC
	PVSs        []PVS
	TESLAMeters []TESLAMeter
	NEURIOs     []NEURIO
	TESLAPVs    []TESLAPV
}

type DeviceCommon struct {
	Din                   string
	PartNumber            string
	SerialNumber          string
	Manufacturer          string
	ComponentParentDin    string
	FirmwareVersion       string
	LastCommunicationTime int64
	EcuType               int32
	Alerts                []string
}

type STSTSM struct {
	Common         DeviceCommon
	STSTSMLocation string
}

type TESYNC struct {
	Common                        DeviceCommon
	ISLANDVL1NMain                float64
	ISLANDFreqL1Main              float64
	ISLANDVL1NLoad                float64
	ISLANDFreqL1Load              float64
	ISLANDPhaseL1MainLoad         float64
	ISLANDVL2NMain                float64
	ISLANDFreqL2Main              float64
	ISLANDVL2NLoad                float64
	ISLANDFreqL2Load              float64
	ISLANDPhaseL2MainLoad         float64
	ISLANDVL3NMain                float64
	ISLANDFreqL3Main              float64
	ISLANDVL3NLoad                float64
	ISLANDFreqL3Load              float64
	ISLANDPhaseL3MainLoad         float64
	ISLANDL1L2PhaseDelta          float64
	ISLANDL1L3PhaseDelta          float64
	ISLANDL2L3PhaseDelta          float64
	ISLANDGridState               string
	ISLANDL1MicrogridOk           bool
	ISLANDL2MicrogridOk           bool
	ISLANDL3MicrogridOk           bool
	ISLANDReadyForSynchronization bool
	ISLANDGridConnected           bool
	SYNCExternallyPowered         bool
	SYNCSiteSwitchEnabled         bool
	METERXCTAInstRealPower        float64
	METERXCTBInstRealPower        float64
	METERXCTCInstRealPower        float64
	METERXCTAInstReactivePower    float64
	METERXCTBInstReactivePower    float64
	METERXCTCInstReactivePower    float64
	METERXLifetimeEnergyImport    float64
	METERXLifetimeEnergyExport    float64
	METERXVL1N                    float64
	METERXVL2N                    float64
	METERXVL3N                    float64
	METERXCTAI                    float64
	METERXCTBI                    float64
	METERXCTCI                    float64
	METERYCTAInstRealPower        float64
	METERYCTBInstRealPower        float64
	METERYCTCInstRealPower        float64
	METERYCTAInstReactivePower    float64
	METERYCTBInstReactivePower    float64
	METERYCTCInstReactivePower    float64
	METERYLifetimeEnergyImport    float64
	METERYLifetimeEnergyExport    float64
	METERYVL1N                    float64
	METERYVL2N                    float64
	METERYVL3N                    float64
	METERYCTAI                    float64
	METERYCTBI                    float64
	METERYCTCI                    float64
}

type TEMSA struct {
	Common                        DeviceCommon
	ISLANDVL1NMain                float64
	ISLANDFreqL1Main              float64
	ISLANDVL1NLoad                float64
	ISLANDFreqL1Load              float64
	ISLANDPhaseL1MainLoad         float64
	ISLANDVL2NMain                float64
	ISLANDFreqL2Main              float64
	ISLANDVL2NLoad                float64
	ISLANDFreqL2Load              float64
	ISLANDPhaseL2MainLoad         float64
	ISLANDVL3NMain                float64
	ISLANDFreqL3Main              float64
	ISLANDVL3NLoad                float64
	ISLANDFreqL3Load              float64
	ISLANDPhaseL3MainLoad         float64
	ISLANDL1L2PhaseDelta          float64
	ISLANDL1L3PhaseDelta          float64
	ISLANDL2L3PhaseDelta          float64
	ISLANDGridState               string
	ISLANDL1MicrogridOk           bool
	ISLANDL2MicrogridOk           bool
	ISLANDL3MicrogridOk           bool
	ISLANDReadyForSynchronization bool
	ISLANDGridConnected           bool
	METERZCTAInstRealPower        float64
	METERZCTBInstRealPower        float64
	METERZCTAInstReactivePower    float64
	METERZCTBInstReactivePower    float64
	METERZLifetimeEnergyNetImport float64
	METERZLifetimeEnergyNetExport float64
	METERZVL1G                    float64
	METERZVL2G                    float64
	METERZCTAI                    float64
	METERZCTBI                    float64
}

type TETHC struct {
	Common         DeviceCommon
	THCState       string
	THCAmbientTemp float64
}

type TEPOD struct {
	Common                  DeviceCommon
	PODNomEnergyToBeCharged float64
	PODNomEnergyRemaining   float64
	PODNomFullPackEnergy    float64
	PODAvailableChargePower float64
	PODAvailableDischgPower float64
	PODState                string
	PODEnableLine           bool
	PODChargeComplete       bool
	PODDischargeComplete    bool
	PODPersistentlyFaulted  bool
	PODPermanentlyFaulted   bool
	PODChargeRequest        bool
	PODActiveHeating        bool
	PODCCVhold              bool
}

type TEPINV struct {
	Common                  DeviceCommon
	PINVEnergyDischarged    float64
	PINVEnergyCharged       float64
	PINVVSplit1             float64
	PINVVSplit2             float64
	PINVPllFrequency        float64
	PINVPllLocked           bool
	PINVPout                float64
	PINVQout                float64
	PINVVout                float64
	PINVFout                float64
	PINVReadyForGridForming bool
	PINVState               string
	PINVGridState           string
	PINVHardwareEnableLine  bool
	PINVPowerLimiter        string
}

type PVAC struct {
	Common                    DeviceCommon
	PVACIout                  float64
	PVACVL1Ground             float64
	PVACVL2Ground             float64
	PVACVHvMinusChassisDC     float64
	PVACPVCurrentA            float64
	PVACPVCurrentB            float64
	PVACPVCurrentC            float64
	PVACPVCurrentD            float64
	PVACPVMeasuredVoltageA    float64
	PVACPVMeasuredVoltageB    float64
	PVACPVMeasuredVoltageC    float64
	PVACPVMeasuredVoltageD    float64
	PVACPVMeasuredPowerA      float64
	PVACPVMeasuredPowerB      float64
	PVACPVMeasuredPowerC      float64
	PVACPVMeasuredPowerD      float64
	PVACLifetimeEnergyPVTotal float64
	PVACVout                  float64
	PVACFout                  float64
	PVACPout                  float64
	PVACQout                  float64
	PVACState                 string
	PVACGridState             string
	PVACInvState              string
	PVACPvStateA              string
	PVACPvStateB              string
	PVACPvStateC              string
	PVACPvStateD              string
	PVIPowerStatusSetpoint    string
}

type PVS struct {
	Common              DeviceCommon
	PVSVLL              float64
	PVSState            string
	PVSSelfTestState    string
	PVSEnableOutput     bool
	PVSStringAConnected bool
	PVSStringBConnected bool
	PVSStringCConnected bool
	PVSStringDConnected bool
}

type TESLAMeter struct {
	Common        DeviceCommon
	MeterLocation []uint32
}

type NEURIO struct {
	Common                 DeviceCommon
	MeterLocation          []uint32
	NEURIOCT0Location      string
	NEURIOCT0InstRealPower float64
}

type TESLAPV struct {
	Common              DeviceCommon
	NameplateRealPowerW uint64
}

func GetVitals(client *http.Client, hostname string, pwa *PowerwallAuth) (*VitalDevices, error) {
	var verbose = false

	body, err := GetPowerwall(client, hostname, "/api/devices/vitals", pwa)

	if err != nil {
		return nil, err
	}

	devices := &pb.DevicesWithVitals{}
	err = proto.Unmarshal(body, devices)
	if err != nil {
		return nil, err
	}
	numd := len(devices.Devices)
	if verbose {
		fmt.Printf("Number of devices %d\n", numd)
	}

	var vd VitalDevices
	for i := 0; i < numd; i++ {
		sccdwv := devices.Devices[i]
		device := sccdwv.Device.Device
		common := &DeviceCommon{}
		common.Din = device.GetDin().GetValue()
		common.PartNumber = device.GetPartNumber().GetValue()
		common.SerialNumber = device.GetSerialNumber().GetValue()
		common.Manufacturer = device.GetManufacturer().GetValue()
		common.ComponentParentDin = device.GetComponentParentDin().GetValue()
		common.FirmwareVersion = device.GetFirmwareVersion().GetValue()
		common.LastCommunicationTime = device.GetLastCommunicationTime().GetSeconds()
		common.EcuType = device.DeviceAttributes.GetTeslaEnergyEcuAttributes().GetEcuType()
		common.Alerts = sccdwv.GetAlerts()

		if strings.Index(common.Din, "STSTSM") == 0 {
			var ststsm STSTSM
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "STSTSM-Location":
					ststsm.STSTSMLocation = vital.GetStringValue()
				default:
					fmt.Printf("Unknown STSTSM DeviceVital.Name %s\n", *vital.Name)
				}
			}
			ststsm.Common = *common
			vd.STSTSM = ststsm
		} else if strings.Index(common.Din, "TESYNC") == 0 {
			var tesync TESYNC
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "ISLAND_VL1N_Main":
					tesync.ISLANDVL1NMain = vital.GetFloatValue()
				case "ISLAND_FreqL1_Main":
					tesync.ISLANDFreqL1Main = vital.GetFloatValue()
				case "ISLAND_VL1N_Load":
					tesync.ISLANDVL1NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL1_Load":
					tesync.ISLANDFreqL1Load = vital.GetFloatValue()
				case "ISLAND_PhaseL1_Main_Load":
					tesync.ISLANDPhaseL1MainLoad = vital.GetFloatValue()
				case "ISLAND_VL2N_Main":
					tesync.ISLANDVL2NMain = vital.GetFloatValue()
				case "ISLAND_FreqL2_Main":
					tesync.ISLANDFreqL2Main = vital.GetFloatValue()
				case "ISLAND_VL2N_Load":
					tesync.ISLANDVL2NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL2_Load":
					tesync.ISLANDFreqL2Load = vital.GetFloatValue()
				case "ISLAND_PhaseL2_Main_Load":
					tesync.ISLANDPhaseL2MainLoad = vital.GetFloatValue()
				case "ISLAND_VL3N_Main":
					tesync.ISLANDVL3NMain = vital.GetFloatValue()
				case "ISLAND_FreqL3_Main":
					tesync.ISLANDFreqL3Main = vital.GetFloatValue()
				case "ISLAND_VL3N_Load":
					tesync.ISLANDVL3NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL3_Load":
					tesync.ISLANDFreqL3Load = vital.GetFloatValue()
				case "ISLAND_PhaseL3_Main_Load":
					tesync.ISLANDPhaseL3MainLoad = vital.GetFloatValue()
				case "ISLAND_L1L2PhaseDelta":
					tesync.ISLANDL1L2PhaseDelta = vital.GetFloatValue()
				case "ISLAND_L1L3PhaseDelta":
					tesync.ISLANDL1L3PhaseDelta = vital.GetFloatValue()
				case "ISLAND_L2L3PhaseDelta":
					tesync.ISLANDL2L3PhaseDelta = vital.GetFloatValue()
				case "ISLAND_GridState":
					tesync.ISLANDGridState = vital.GetStringValue()
				case "ISLAND_L1MicrogridOk":
					tesync.ISLANDL1MicrogridOk = vital.GetBoolValue()
				case "ISLAND_L2MicrogridOk":
					tesync.ISLANDL2MicrogridOk = vital.GetBoolValue()
				case "ISLAND_L3MicrogridOk":
					tesync.ISLANDL3MicrogridOk = vital.GetBoolValue()
				case "ISLAND_ReadyForSynchronization":
					tesync.ISLANDReadyForSynchronization = vital.GetBoolValue()
				case "ISLAND_GridConnected":
					tesync.ISLANDGridConnected = vital.GetBoolValue()
				case "SYNC_ExternallyPowered":
					tesync.SYNCExternallyPowered = vital.GetBoolValue()
				case "SYNC_SiteSwitchEnabled":
					tesync.SYNCSiteSwitchEnabled = vital.GetBoolValue()
				case "METER_X_CTA_InstRealPower":
					tesync.METERXCTAInstRealPower = vital.GetFloatValue()
				case "METER_X_CTB_InstRealPower":
					tesync.METERXCTBInstRealPower = vital.GetFloatValue()
				case "METER_X_CTC_InstRealPower":
					tesync.METERXCTCInstRealPower = vital.GetFloatValue()
				case "METER_X_CTA_InstReactivePower":
					tesync.METERXCTAInstReactivePower = vital.GetFloatValue()
				case "METER_X_CTB_InstReactivePower":
					tesync.METERXCTBInstReactivePower = vital.GetFloatValue()
				case "METER_X_CTC_InstReactivePower":
					tesync.METERXCTCInstReactivePower = vital.GetFloatValue()
				case "METER_X_LifetimeEnergyImport":
					tesync.METERXLifetimeEnergyImport = vital.GetFloatValue()
				case "METER_X_LifetimeEnergyExport":
					tesync.METERXLifetimeEnergyExport = vital.GetFloatValue()
				case "METER_X_VL1N":
					tesync.METERXVL1N = vital.GetFloatValue()
				case "METER_X_VL2N":
					tesync.METERXVL2N = vital.GetFloatValue()
				case "METER_X_VL3N":
					tesync.METERXVL3N = vital.GetFloatValue()
				case "METER_X_CTA_I":
					tesync.METERXCTAI = vital.GetFloatValue()
				case "METER_X_CTB_I":
					tesync.METERXCTBI = vital.GetFloatValue()
				case "METER_X_CTC_I":
					tesync.METERXCTCI = vital.GetFloatValue()
				case "METER_Y_CTA_InstRealPower":
					tesync.METERYCTAInstRealPower = vital.GetFloatValue()
				case "METER_Y_CTB_InstRealPower":
					tesync.METERYCTBInstRealPower = vital.GetFloatValue()
				case "METER_Y_CTC_InstRealPower":
					tesync.METERYCTCInstRealPower = vital.GetFloatValue()
				case "METER_Y_CTA_InstReactivePower":
					tesync.METERYCTAInstReactivePower = vital.GetFloatValue()
				case "METER_Y_CTB_InstReactivePower":
					tesync.METERYCTBInstReactivePower = vital.GetFloatValue()
				case "METER_Y_CTC_InstReactivePower":
					tesync.METERYCTCInstReactivePower = vital.GetFloatValue()
				case "METER_Y_LifetimeEnergyImport":
					tesync.METERYLifetimeEnergyImport = vital.GetFloatValue()
				case "METER_Y_LifetimeEnergyExport":
					tesync.METERYLifetimeEnergyExport = vital.GetFloatValue()
				case "METER_Y_VL1N":
					tesync.METERYVL1N = vital.GetFloatValue()
				case "METER_Y_VL2N":
					tesync.METERYVL2N = vital.GetFloatValue()
				case "METER_Y_VL3N":
					tesync.METERYVL3N = vital.GetFloatValue()
				case "METER_Y_CTA_I":
					tesync.METERYCTAI = vital.GetFloatValue()
				case "METER_Y_CTB_I":
					tesync.METERYCTBI = vital.GetFloatValue()
				case "METER_Y_CTC_I":
					tesync.METERYCTCI = vital.GetFloatValue()
				default:
					fmt.Printf("Unknown TESYNC DeviceVital.Name %s\n", *vital.Name)
				}
			}
			tesync.Common = *common
			vd.TESYNC = tesync
		} else if strings.Index(common.Din, "TEMSA") == 0 {
			var temsa TEMSA
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "ISLAND_VL1N_Main":
					temsa.ISLANDVL1NMain = vital.GetFloatValue()
				case "ISLAND_FreqL1_Main":
					temsa.ISLANDFreqL1Main = vital.GetFloatValue()
				case "ISLAND_VL1N_Load":
					temsa.ISLANDVL1NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL1_Load":
					temsa.ISLANDFreqL1Load = vital.GetFloatValue()
				case "ISLAND_PhaseL1_Main_Load":
					temsa.ISLANDPhaseL1MainLoad = vital.GetFloatValue()
				case "ISLAND_VL2N_Main":
					temsa.ISLANDVL2NMain = vital.GetFloatValue()
				case "ISLAND_FreqL2_Main":
					temsa.ISLANDFreqL2Main = vital.GetFloatValue()
				case "ISLAND_VL2N_Load":
					temsa.ISLANDVL2NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL2_Load":
					temsa.ISLANDFreqL2Load = vital.GetFloatValue()
				case "ISLAND_PhaseL2_Main_Load":
					temsa.ISLANDPhaseL2MainLoad = vital.GetFloatValue()
				case "ISLAND_VL3N_Main":
					temsa.ISLANDVL3NMain = vital.GetFloatValue()
				case "ISLAND_FreqL3_Main":
					temsa.ISLANDFreqL3Main = vital.GetFloatValue()
				case "ISLAND_VL3N_Load":
					temsa.ISLANDVL3NLoad = vital.GetFloatValue()
				case "ISLAND_FreqL3_Load":
					temsa.ISLANDFreqL3Load = vital.GetFloatValue()
				case "ISLAND_PhaseL3_Main_Load":
					temsa.ISLANDPhaseL3MainLoad = vital.GetFloatValue()
				case "ISLAND_L1L2PhaseDelta":
					temsa.ISLANDL1L2PhaseDelta = vital.GetFloatValue()
				case "ISLAND_L1L3PhaseDelta":
					temsa.ISLANDL1L3PhaseDelta = vital.GetFloatValue()
				case "ISLAND_L2L3PhaseDelta":
					temsa.ISLANDL2L3PhaseDelta = vital.GetFloatValue()
				case "ISLAND_GridState":
					temsa.ISLANDGridState = vital.GetStringValue()
				case "ISLAND_L1MicrogridOk":
					temsa.ISLANDL1MicrogridOk = vital.GetBoolValue()
				case "ISLAND_L2MicrogridOk":
					temsa.ISLANDL2MicrogridOk = vital.GetBoolValue()
				case "ISLAND_L3MicrogridOk":
					temsa.ISLANDL3MicrogridOk = vital.GetBoolValue()
				case "ISLAND_ReadyForSynchronization":
					temsa.ISLANDReadyForSynchronization = vital.GetBoolValue()
				case "ISLAND_GridConnected":
					temsa.ISLANDGridConnected = vital.GetBoolValue()
				case "METER_Z_CTA_InstRealPower":
					temsa.METERZCTAInstRealPower = vital.GetFloatValue()
				case "METER_Z_CTB_InstRealPower":
					temsa.METERZCTBInstRealPower = vital.GetFloatValue()
				case "METER_Z_CTA_InstReactivePower":
					temsa.METERZCTAInstReactivePower = vital.GetFloatValue()
				case "METER_Z_CTB_InstReactivePower":
					temsa.METERZCTBInstReactivePower = vital.GetFloatValue()
				case "METER_Z_LifetimeEnergyNetImport":
					temsa.METERZLifetimeEnergyNetImport = vital.GetFloatValue()
				case "METER_Z_LifetimeEnergyNetExport":
					temsa.METERZLifetimeEnergyNetExport = vital.GetFloatValue()
				case "METER_Z_VL1G":
					temsa.METERZVL1G = vital.GetFloatValue()
				case "METER_Z_VL2G":
					temsa.METERZVL2G = vital.GetFloatValue()
				case "METER_Z_CTA_I":
					temsa.METERZCTAI = vital.GetFloatValue()
				case "METER_Z_CTB_I":
					temsa.METERZCTBI = vital.GetFloatValue()
				default:
					fmt.Printf("Unknown DeviceVital.Name %s\n", *vital.Name)
				}
			}
			temsa.Common = *common
			vd.TEMSA = temsa
		} else if strings.Index(common.Din, "TETHC") == 0 {
			var tethc TETHC
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "THC_State":
					tethc.THCState = vital.GetStringValue()
				case "THC_AmbientTemp":
					tethc.THCAmbientTemp = vital.GetFloatValue()
				default:
					fmt.Printf("Unknown TETHC DeviceVital.Name %s\n", *vital.Name)
				}
			}
			tethc.Common = *common
			vd.TETHCs = append(vd.TETHCs, tethc)
		} else if strings.Index(common.Din, "TEPOD") == 0 {
			var tepod TEPOD
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "POD_nom_energy_to_be_charged":
					tepod.PODNomEnergyToBeCharged = vital.GetFloatValue()
				case "POD_nom_energy_remaining":
					tepod.PODNomEnergyRemaining = vital.GetFloatValue()
				case "POD_nom_full_pack_energy":
					tepod.PODNomFullPackEnergy = vital.GetFloatValue()
				case "POD_available_charge_power":
					tepod.PODAvailableChargePower = vital.GetFloatValue()
				case "POD_available_dischg_power":
					tepod.PODAvailableDischgPower = vital.GetFloatValue()
				case "POD_state":
					tepod.PODState = vital.GetStringValue()
				case "POD_enable_line":
					tepod.PODEnableLine = vital.GetBoolValue()
				case "POD_ChargeComplete":
					tepod.PODChargeComplete = vital.GetBoolValue()
				case "POD_DischargeComplete":
					tepod.PODDischargeComplete = vital.GetBoolValue()
				case "POD_PersistentlyFaulted":
					tepod.PODPersistentlyFaulted = vital.GetBoolValue()
				case "POD_PermanentlyFaulted":
					tepod.PODPermanentlyFaulted = vital.GetBoolValue()
				case "POD_ChargeRequest":
					tepod.PODChargeRequest = vital.GetBoolValue()
				case "POD_ActiveHeating":
					tepod.PODActiveHeating = vital.GetBoolValue()
				case "POD_CCVhold":
					tepod.PODCCVhold = vital.GetBoolValue()

				default:
					fmt.Printf("Unknown TEPOD DeviceVital.Name %s\n", *vital.Name)
				}
			}
			tepod.Common = *common
			vd.TEPODs = append(vd.TEPODs, tepod)
		} else if strings.Index(common.Din, "TEPINV") == 0 {
			var tepinv TEPINV
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "PINV_EnergyDischarged":
					tepinv.PINVEnergyDischarged = vital.GetFloatValue()
				case "PINV_EnergyCharged":
					tepinv.PINVEnergyCharged = vital.GetFloatValue()
				case "PINV_VSplit1":
					tepinv.PINVVSplit1 = vital.GetFloatValue()
				case "PINV_VSplit2":
					tepinv.PINVVSplit2 = vital.GetFloatValue()
				case "PINV_PllFrequency":
					tepinv.PINVPllFrequency = vital.GetFloatValue()
				case "PINV_PllLocked":
					tepinv.PINVPllLocked = vital.GetBoolValue()
				case "PINV_Pout":
					tepinv.PINVPout = vital.GetFloatValue()
				case "PINV_Qout":
					tepinv.PINVQout = vital.GetFloatValue()
				case "PINV_Vout":
					tepinv.PINVVout = vital.GetFloatValue()
				case "PINV_Fout":
					tepinv.PINVFout = vital.GetFloatValue()
				case "PINV_ReadyForGridForming":
					tepinv.PINVReadyForGridForming = vital.GetBoolValue()
				case "PINV_State":
					tepinv.PINVState = vital.GetStringValue()
				case "PINV_GridState":
					tepinv.PINVGridState = vital.GetStringValue()
				case "PINV_HardwareEnableLine":
					tepinv.PINVHardwareEnableLine = vital.GetBoolValue()
				case "PINV_PowerLimiter":
					tepinv.PINVPowerLimiter = vital.GetStringValue()
				default:
					fmt.Printf("Unknown DeviceVital.Name %s\n", *vital.Name)
				}
			}
			tepinv.Common = *common
			vd.TEPINVs = append(vd.TEPINVs, tepinv)
		} else if strings.Index(common.Din, "PVAC") == 0 {
			var pvac PVAC
			numv := len(sccdwv.Vitals)
			if verbose {
				fmt.Printf("Number of vitals %d\n", numv)
			}
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "PVAC_Iout":
					pvac.PVACIout = vital.GetFloatValue()
				case "PVAC_VL1Ground":
					pvac.PVACVL1Ground = vital.GetFloatValue()
				case "PVAC_VL2Ground":
					pvac.PVACVL2Ground = vital.GetFloatValue()
				case "PVAC_VHvMinusChassisDC":
					pvac.PVACVHvMinusChassisDC = vital.GetFloatValue()
				case "PVAC_PVCurrent_A":
					pvac.PVACPVCurrentA = vital.GetFloatValue()
				case "PVAC_PVCurrent_B":
					pvac.PVACPVCurrentB = vital.GetFloatValue()
				case "PVAC_PVCurrent_C":
					pvac.PVACPVCurrentC = vital.GetFloatValue()
				case "PVAC_PVCurrent_D":
					pvac.PVACPVCurrentD = vital.GetFloatValue()
				case "PVAC_PVMeasuredVoltage_A":
					pvac.PVACPVMeasuredVoltageA = vital.GetFloatValue()
				case "PVAC_PVMeasuredVoltage_B":
					pvac.PVACPVMeasuredVoltageB = vital.GetFloatValue()
				case "PVAC_PVMeasuredVoltage_C":
					pvac.PVACPVMeasuredVoltageC = vital.GetFloatValue()
				case "PVAC_PVMeasuredVoltage_D":
					pvac.PVACPVMeasuredVoltageD = vital.GetFloatValue()
				case "PVAC_PVMeasuredPower_A":
					pvac.PVACPVMeasuredPowerA = vital.GetFloatValue()
				case "PVAC_PVMeasuredPower_B":
					pvac.PVACPVMeasuredPowerB = vital.GetFloatValue()
				case "PVAC_PVMeasuredPower_C":
					pvac.PVACPVMeasuredPowerC = vital.GetFloatValue()
				case "PVAC_PVMeasuredPower_D":
					pvac.PVACPVMeasuredPowerD = vital.GetFloatValue()
				case "PVAC_LifetimeEnergyPV_Total":
					pvac.PVACLifetimeEnergyPVTotal = vital.GetFloatValue()
				case "PVAC_Vout":
					pvac.PVACVout = vital.GetFloatValue()
				case "PVAC_Fout":
					pvac.PVACFout = vital.GetFloatValue()
				case "PVAC_Pout":
					pvac.PVACPout = vital.GetFloatValue()
				case "PVAC_Qout":
					pvac.PVACQout = vital.GetFloatValue()
				case "PVAC_State":
					pvac.PVACState = vital.GetStringValue()
				case "PVAC_GridState":
					pvac.PVACGridState = vital.GetStringValue()
				case "PVAC_InvState":
					pvac.PVACInvState = vital.GetStringValue()
				case "PVAC_PvState_A":
					pvac.PVACPvStateA = vital.GetStringValue()
				case "PVAC_PvState_B":
					pvac.PVACPvStateB = vital.GetStringValue()
				case "PVAC_PvState_C":
					pvac.PVACPvStateC = vital.GetStringValue()
				case "PVAC_PvState_D":
					pvac.PVACPvStateD = vital.GetStringValue()
				case "PVI-PowerStatusSetpoint":
					pvac.PVIPowerStatusSetpoint = vital.GetStringValue()
				default:
					fmt.Printf("Unknown DeviceVital.Name %s\n", *vital.Name)
				}
			}
			pvac.Common = *common
			vd.PVACs = append(vd.PVACs, pvac)
		} else if strings.Index(common.Din, "PVS") == 0 {
			var pvs PVS
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "PVS_vLL":
					pvs.PVSVLL = vital.GetFloatValue()
				case "PVS_State":
					pvs.PVSState = vital.GetStringValue()
				case "PVS_SelfTestState":
					pvs.PVSSelfTestState = vital.GetStringValue()
				case "PVS_EnableOutput":
					pvs.PVSEnableOutput = vital.GetBoolValue()
				case "PVS_StringA_Connected":
					pvs.PVSStringAConnected = vital.GetBoolValue()
				case "PVS_StringB_Connected":
					pvs.PVSStringBConnected = vital.GetBoolValue()
				case "PVS_StringC_Connected":
					pvs.PVSStringCConnected = vital.GetBoolValue()
				case "PVS_StringD_Connected":
					pvs.PVSStringDConnected = vital.GetBoolValue()
				default:
					fmt.Printf("Unknown TEPINV DeviceVital.Name %s\n", *vital.Name)
				}
			}
			pvs.Common = *common
			vd.PVSs = append(vd.PVSs, pvs)
		} else if strings.Index(common.Din, "TESLA") == 0 {
			// need to check for meter vs pv cases
			ma := device.DeviceAttributes.GetMeterAttributes()
			pvia := device.DeviceAttributes.GetPvInverterAttributes()
			if ma != nil {
				var tesla TESLAMeter
				tesla.MeterLocation = ma.MeterLocation
				numv := len(sccdwv.Vitals)
				for j := 0; j < numv; j++ {
					vital := sccdwv.Vitals[j]
					switch *vital.Name {
					default:
						fmt.Printf("Unknown TESLA Meter DeviceVital.Name %s\n", *vital.Name)
					}
				}
				tesla.Common = *common
				vd.TESLAMeters = append(vd.TESLAMeters, tesla)
			} else if pvia != nil {
				var tesla TESLAPV
				tesla.NameplateRealPowerW = pvia.NameplateRealPowerW
				numv := len(sccdwv.Vitals)
				for j := 0; j < numv; j++ {
					vital := sccdwv.Vitals[j]
					switch *vital.Name {
					default:
						fmt.Printf("Unknown TESLA PV DeviceVital.Name %s\n", *vital.Name)
					}
				}
				tesla.Common = *common
				vd.TESLAPVs = append(vd.TESLAPVs, tesla)
			} else {
				fmt.Printf("Unknown TESLA device in vitals\n")
			}
		} else if strings.Index(common.Din, "NEURIO") == 0 {
			var neurio NEURIO
			numv := len(sccdwv.Vitals)
			for j := 0; j < numv; j++ {
				vital := sccdwv.Vitals[j]
				switch *vital.Name {
				case "NEURIO_CT0_Location":
					neurio.NEURIOCT0Location = vital.GetStringValue()
				case "NEURIO_CT0_InstRealPower":
					neurio.NEURIOCT0InstRealPower = vital.GetFloatValue()
				default:
					fmt.Printf("Unknown NEURIO DeviceVital.Name %s\n", *vital.Name)
				}
			}
			neurio.Common = *common
			ma := device.DeviceAttributes.GetMeterAttributes()
			if ma != nil {
				neurio.MeterLocation = ma.MeterLocation
			}
			vd.NEURIOs = append(vd.NEURIOs, neurio)
		}
	}

	if verbose {
		json := protojson.Format(devices)
		fmt.Printf("Resp JSON %s\n", json)
	}

	return &vd, nil
}

// GetPowerwall performs a GET request to a local Tesla Powerwall gateway.
// It doesn't do authentication yet.
func GetPowerwall(client *http.Client, hostname string, endpoint string, pwa *PowerwallAuth) ([]byte, error) {

	var verbose = false

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

	if pwa != nil {
		req.Header.Add("Cookie", "AuthCookie="+pwa.Token)
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

// GetPowerwallAuth gets a token (plus some other stuff) for authentication
// on a local Powerwall gateway
func GetPowerwallAuth(client *http.Client, hostname string, email string, password string) (*PowerwallAuth, error) {

	type PowerwallLogin struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var pl PowerwallLogin
	var pa PowerwallAuth

	var verbose = false

	// Figure out the correct endpoint
	var url = "https://" + hostname + "/api/login/Basic"
	if verbose {
		fmt.Printf("URL: %s\n", url)
	}

	// JSON payload with login info
	pl.Username = "customer"
	pl.Email = email
	pl.Password = password
	payload, err := json.Marshal(pl)

	// Set up POST
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

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

	// Parse response, get auth token
	err = json.Unmarshal(body, &pa)
	if err != nil {
		return nil, err
	}

	// Parse timestamp
	pa.Timestamp, err = time.Parse(time.RFC3339Nano, pa.LoginTime)

	return &pa, nil
}
