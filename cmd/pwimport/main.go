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
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	influxClient "github.com/influxdata/influxdb1-client/v2" // too many things called "client"
)

// InfluxDB parameters
var InfluxUrl string
var InfluxDb string
var InfluxMeasurement string

var hostname string

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

func main() {
	var verbose = false
	var pollTime = 1.0

	// Seed random number generator, for semi-random polling interval
	rand.Seed(time.Now().UTC().UnixNano())

	// Command-line arguments
	flag.StringVar(&InfluxUrl, "influx-url", "http://localhost:8086",
		"Influx database server URL")
	flag.StringVar(&InfluxDb, "influx-database", "tesla",
		"Influx database name")
	flag.StringVar(&InfluxMeasurement, "influx-measurement", "powerwall",
		"Influx measurement name")
	flag.StringVar(&hostname, "hostname", "teg", "Powerwall gateway hostname")
	flag.Float64Var(&pollTime, "poll", 1.0, "Polling interval (seconds)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	// Parse command-line arguments
	flag.Parse()

	/*
		// Get cached Tesla authentication token
		token, err := gotesla.LoadCachedToken()
		if err != nil {
			fmt.Println(err)
			return
		}
		if verbose {
			fmt.Printf("%+v\n", token)
		}
	*/

	// Don't verify TLS certs...
	tls := &tls.Config{InsecureSkipVerify: true}

	// Get TLS transport
	tr := &http.Transport{TLSClientConfig: tls}

	// Make an HTTPS client
	client := &http.Client{Transport: tr}

	// Get a new HTTP client for InfluxDB
	dbClient, err := influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr: InfluxUrl,
	})
	if err != nil {
		log.Fatalf("NewHTTPClient: %v\n", err)
	}
	defer dbClient.Close()

	// Loop forever...
	for {

		// Get aggregate meters...these give us power, current,
		// and voltage for the grid, solar, Powerwall battery, and
		// house load.
		ma, err := gotesla.GetMeterAggregate(client, hostname)
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
		soe, err := gotesla.GetSoe(client, hostname)
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
		gs, err := gotesla.GetGridStatus(client, hostname)
		if err != nil {
			log.Printf("GetGridStatus: %v\n", err)
			continue
		}
		if verbose {
			log.Printf("Grid Status: %v\n", gs)
		}

		// Take a timestamp for any data that's not already
		// timestamped
		now := time.Now().Round(0)

		// Batch of data points.  We'll have one point each for
		// the grid (site), Powerwall (battery), solar,
		// and house (load).  Each of those will be timestamped
		// from the last_communication_time field, and will
		// contain (most of) the fields from the Meter structure.
		// Another point will hold the SOE and grid status.
		bp, err := influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
			Database:  InfluxDb,
			Precision: "us",
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

		// Create the point with SOE and grid status
		{
			tags := map[string]string{}
			fields := map[string]interface{}{
				"soe":         soe,
				"grid_status": int(gs),
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

		// Write data points in the batch
		err = dbClient.Write(bp)
		if err != nil {
			log.Printf("Write: %v\n", err)
			continue
		}

		// Sleep for awhile...
		if verbose {
			fmt.Printf("Sleeping for %f\n\n", pollTime)
		}
		time.Sleep(time.Duration(pollTime) * time.Second)

	}
}
