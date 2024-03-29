//
// Copyright (C) 2020-2021 Bruce A. Mah.
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
	"github.com/bmah888/gotesla"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	influxClient "github.com/influxdata/influxdb1-client/v2" // too many things called "client"
)

// InfluxURL is the URL to the InfluxDB server
var InfluxURL string

// InfluxDb is the database name
var InfluxDb string

// InfluxMeasurement is the name of the InfluxDB measurement
var InfluxMeasurement string

var hostname string
var email string
var password string

// makeMeterPoint constructs an InfluxDB measurement point from a
// Meter structure.
func makeMeterPoint(measurement string, meterName string, meter *gotesla.Meter) (*influxClient.Point, error) {
	tags := map[string]string{
		"meter": meterName,
	}
	fields := map[string]interface{}{
		"instant_power":           int(meter.InstantPower),
		"instant_power_min0":      int(math.Min(0.0, meter.InstantPower)),
		"instant_power_max0":      int(math.Max(0.0, meter.InstantPower)),
		"frequency":               meter.Frequency,
		"energy_exported":         int(meter.EnergyExported),
		"energy_imported":         int(meter.EnergyImported),
		"instant_average_voltage": int(meter.InstantAverageVoltage),
		"instant_total_current":   int(meter.InstantTotalCurrent),
	}
	timestamp, err := time.Parse(time.RFC3339Nano, meter.LastCommunicationTime)
	if err != nil {
		return nil, err // XXX fix error
	}

	pt, err := influxClient.NewPoint(
		measurement,
		tags,
		fields,
		timestamp,
	)
	if err != nil {
		return nil, err // XXX fix error
	}

	return pt, nil
}

// makeFullPackEnergyPoint constructs a measurement point from a
// BatteryBlock structures from the system_status API call
func makeFullPackEnergyPoint(measurement string, now time.Time, batt gotesla.BatteryBlock) (*influxClient.Point, error) {
	// Pull the various points out of the BatteryBlock and feed to
	// a lower-level version of this function.
	return makeFullPackEnergyPoint2(measurement,
		now,
		batt.PackageSerialNumber,
		batt.NominalFullPackEnergy,
		batt.NominalEnergyRemaining,
		batt.EnergyCharged,
		batt.EnergyDischarged)
}

// makeFullPackEnergyPoint2 constructs a measurement point from
// discrete values. Useful for synthesizing data points for an entire
// Powerwall system.
func makeFullPackEnergyPoint2(measurement string,
	now time.Time,
	packageSerialNumber string,
	nominalFullPackEnergy int,
	nominalEnergyRemaining int,
	energyCharged int,
	energyDischarged int) (*influxClient.Point, error) {

	tags := map[string]string{
		"PackageSerialNumber": packageSerialNumber,
	}
	fields := map[string]interface{}{
		"nominal_full_pack_energy": nominalFullPackEnergy,
		"nominal_energy_remaining": nominalEnergyRemaining,
		"energy_charged":           energyCharged,
		"energy_discharged":        energyDischarged,
	}
	pt, err := influxClient.NewPoint(
		measurement,
		tags,
		fields,
		now)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

func main() {
	var verbose bool
	var pollTime float64
	var refreshTime float64

	// Seed random number generator, for semi-random polling interval
	rand.Seed(time.Now().UTC().UnixNano())

	// Command-line arguments
	flag.StringVar(&InfluxURL, "influx-url", "http://localhost:8086",
		"Influx database server URL")
	flag.StringVar(&InfluxDb, "influx-database", "tesla",
		"Influx database name")
	flag.StringVar(&InfluxMeasurement, "influx-measurement", "powerwall",
		"Influx measurement name")
	flag.StringVar(&hostname, "hostname", "teg", "Powerwall gateway hostname")
	flag.StringVar(&email, "email", "", "Email address for login")
	flag.StringVar(&password, "password", "", "Password for login")
	flag.Float64Var(&pollTime, "poll", 10.0, "Polling interval (seconds)")
	flag.Float64Var(&refreshTime, "refresh", 3600.0, "Token refresh interval (seconds)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	// Parse command-line arguments
	flag.Parse()

	// Don't verify TLS certs...
	tls := &tls.Config{InsecureSkipVerify: true}

	// Get TLS transport
	tr := &http.Transport{TLSClientConfig: tls}

	// Make an HTTPS client
	client := &http.Client{Transport: tr}

	var err error

	// Get an authentication token
	var pwa *gotesla.PowerwallAuth
	if email != "" && password != "" {
		pwa, err = gotesla.GetPowerwallAuth(client, hostname, email, password)
		if err != nil {
			log.Fatalf("PowerwallAuth: %v\n", err)
		}
	}

	// Maybe print out some stuff from the token
	if verbose {
		if pwa != nil {
			fmt.Printf("email %s\n", pwa.Email)
			fmt.Printf("token %s\n", pwa.Token)
			fmt.Printf("timestamp %s\n", pwa.Timestamp.Format(time.UnixDate))
		}
	}

	// Get a new HTTP client for InfluxDB
	dbClient, err := influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr: InfluxURL,
	})
	if err != nil {
		log.Fatalf("NewHTTPClient: %v\n", err)
	}
	defer dbClient.Close()

	// Loop forever...
	for ; ; time.Sleep(time.Duration(pollTime) * time.Second) {

		// Get aggregate meters...these give us power, current,
		// and voltage for the grid, solar, Powerwall battery, and
		// house load.
		ma, err := gotesla.GetMeterAggregate(client, hostname, pwa)
		if err != nil {
			log.Printf("GetMeterAggregate: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("%+v\n", ma)
		}

		// Get SOE (state of energy) of the Powerwall battery,
		// it's a float percentage from 0-100 for the entire
		// system (potentially multiple batteries).
		soe, err := gotesla.GetSoe(client, hostname, pwa)
		if err != nil {
			log.Printf("GetSoe: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("SOE: %f\n", soe)
		}

		// Get the grid status
		// We define that within the gotesla package as a
		// scalar (see the declaration of GridStatus), but note
		// that it needs to be converted to an int eventually.
		gs, err := gotesla.GetGridStatus(client, hostname, pwa)
		if err != nil {
			log.Printf("GetGridStatus: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("Grid Status: %v\n", gs)
		}

		// Get the sitemaster status.  This is mostly useful
		// for the Powerwall start/stop state and the connected to
		// Tesla state.
		sm, err := gotesla.GetSiteMaster(client, hostname, pwa)
		if err != nil {
			log.Printf("GetSiteMaster: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("SiteMaster: %v\n", sm)
		}

		// Get the system status, for the battery capacity
		sysstat, err := gotesla.GetSystemStatus(client, hostname, pwa)
		if err != nil {
			log.Printf("GetSystemStatus: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("SystemStatus: %v\n", sysstat)
		}

		// Take a timestamp for any data that's not already
		// timestamped
		now := time.Now().Round(0)

		// Batch of data points.  We'll have one point each for
		// the grid (site), Powerwall (battery), solar,
		// and house (load).  Each of those will be timestamped
		// from the last_communication_time field, and will
		// contain (most of) the fields from the Meter structure.
		// Another point will hold the SOE, grid status, running
		// and connection.
		bp, err := influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
			Database:  InfluxDb,
			Precision: "s",
		})
		if err != nil {
			log.Printf("NewBatchPoints: %v\n", err)
			continue
		}

		// Use a helper function to create the various points
		p1, err := makeMeterPoint(InfluxMeasurement, "site", &(ma.Site))
		if err != nil {
			log.Printf("makeMeterPoint(site): %v\n", err)
			continue
		}
		if verbose {
			fmt.Printf("site: %+v\n", p1)
		}
		bp.AddPoint(p1)

		p2, err := makeMeterPoint(InfluxMeasurement, "battery", &(ma.Battery))
		if err != nil {
			log.Printf("makeMeterPoint(battery): %v\n", err)
			continue
		}
		if verbose {
			fmt.Printf("battery: %+v\n", p2)
		}
		bp.AddPoint(p2)

		p3, err := makeMeterPoint(InfluxMeasurement, "load", &(ma.Load))
		if err != nil {
			log.Printf("makeMeterPoint(load): %v\n", err)
			continue
		}
		if verbose {
			fmt.Printf("load: %+v\n", p3)
		}
		bp.AddPoint(p3)

		p4, err := makeMeterPoint(InfluxMeasurement, "solar", &(ma.Solar))
		if err != nil {
			log.Printf("makeMeterPoint(solar): %v\n", err)
			continue
		}
		if verbose {
			fmt.Printf("solar: %+v\n", p4)
		}
		bp.AddPoint(p4)

		// Create the point with SOE, grid status, and other status variables
		{
			tags := map[string]string{}

			// A couple of booleans we want to record need to
			// be converted to integers first because Grafana
			// has difficulty dealing with graphing boolean
			// values.
			var running, connectedToTesla int8
			if sm.Running {
				running = 1
			}
			if sm.ConnectedToTesla {
				connectedToTesla = 1
			}

			// Convert from API SOE values to the values displayed
			// in the Tesla mobile app, so the values stored to
			// the database match the app.  It's a linear scaling
			// described in (e.g.):
			// https://teslamotorsclub.com/tmc/posts/4360544/
			// https://teslamotorsclub.com/tmc/posts/4360595/
			soe = (soe - 5) / 0.95
			if verbose {
				log.Printf("Scaled SOE: %f\n", soe)
			}

			fields := map[string]interface{}{
				"soe":                soe,
				"grid_status":        int(gs),
				"running":            running,
				"connected_to_tesla": connectedToTesla,
			}

			pt, err := influxClient.NewPoint(
				InfluxMeasurement,
				tags,
				fields,
				now,
			)
			if err != nil {
				log.Printf("NewPoint: %v\n", err)
				continue
			}
			bp.AddPoint(pt)
		}

		// Create battery and sum points from system status
		var i int
		var totalCharged, totalDischarged int
		for i = 0; i < sysstat.AvailableBlocks; i++ {
			battp, err := makeFullPackEnergyPoint(InfluxMeasurement, now, sysstat.BatteryBlocks[i])
			if err != nil {
				log.Printf("makeFullEnergyPackPoint: %v\n", err)
				continue
			}
			if verbose {
				fmt.Printf("batt: %+v\n", battp)
			}

			// For computing system total charge/discharge energy
			totalCharged += sysstat.BatteryBlocks[i].EnergyCharged
			totalDischarged += sysstat.BatteryBlocks[i].EnergyDischarged

			bp.AddPoint(battp)
		}

		// System total
		sysp, err := makeFullPackEnergyPoint2(InfluxMeasurement,
			now,
			"total",
			sysstat.NominalFullPackEnergy,
			sysstat.NominalEnergyRemaining,
			totalCharged,
			totalDischarged)
		if err != nil {
			log.Printf("makeFullPackEnergyPoint2: %v\n", err)
			continue
		}
		if verbose {
			fmt.Printf("sys: %+v\n", sysp)
		}
		bp.AddPoint(sysp)

		// Write data points in the batch
		err = dbClient.Write(bp)
		if err != nil {
			log.Printf("Write: %v\n", err)
		}

		// If we needed to authenticate, then the authentication
		// token might need a refresh. The tokens don't have
		// explicit expiration times, so we have to refresh
		// at some hopefully short enough interval.
		if pwa != nil {

			// How old is the token?
			tokenAge := time.Since(pwa.Timestamp)
			if verbose {
				fmt.Printf("tokenAge %v\n", tokenAge.String())
			}

			if tokenAge.Seconds() > refreshTime {
				if verbose {
					fmt.Printf("Reauthenticate token\n")
				}
				if email != "" && password != "" {
					pwa, _ = gotesla.GetPowerwallAuth(client, hostname, email, password)
				}
			}
		}
	}
}
