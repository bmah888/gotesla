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
	"math/rand"
	"net/http"
	"time"

	influxClient "github.com/influxdata/influxdb1-client/v2" // too many things called "client"
)

// InfluxDB parameters
var InfluxUrl string
var InfluxDb string
var InfluxMeasurement string

func main() {
	var verbose bool = false

	// Seed random number generator, for semi-random polling interval
	rand.Seed(time.Now().UTC().UnixNano())

	// Command-line arguments
	flag.StringVar(&InfluxUrl, "influx-url", "http://localhost:8086",
		"Influx database server URL")
	flag.StringVar(&InfluxDb, "influx-database", "tesla",
		"Influx database name")
	flag.StringVar(&InfluxMeasurement, "influx-measurement", "chargers",
		"Influx measurement name")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

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
	vr, err := gotesla.GetVehicles(client, token)
	if err != nil {
		fmt.Println(err)
		return
	}

	if verbose {
		fmt.Printf("%d vehicles retrieved\n", vr.Count)
	}

	// Get a new HTTP client for InfluxDB
	dbClient, err := influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr: InfluxUrl,
	})
	if err != nil {
		fmt.Println(err)
	}
	defer dbClient.Close()

	for {

		for _, v := range vr.Response {
			if verbose {
				fmt.Printf("Vehicle: id %d VIN %s\n", v.Id, v.Vin)
			}

			nc, err := gotesla.GetNearbyChargers(client, token, v.Id)
			if err != nil {
				fmt.Println(err)
				return
			}
			timeCongestion := time.Unix(int64(nc.Response.CongestionSyncTimeUtcSecs), 0)
			timeStamp := time.Unix(int64(nc.Response.Timestamp/1000), 0)
			if verbose {
				fmt.Printf("CongestionSyncTimeUtcSecs: %s\n", timeCongestion.Format(time.RFC3339))
				fmt.Printf("TimeStamp: %s\n", timeStamp.Format(time.RFC3339))
			}

			// Make a batch of points
			bp, err := influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
				Database:  InfluxDb,
				Precision: "us",
			})
			if err != nil {
				fmt.Println(err)
				return
			}

			// For each Supercharger, make up a data point
			// and add it to the Influx batch.
			for _, suc := range nc.Response.Superchargers {

				tags := map[string]string{
					"type": suc.Type,
					"name": suc.Name,
				}
				fields := map[string]interface{}{
					"available_stalls": suc.AvailableStalls,
					"total_stalls":     suc.TotalStalls,
					"site_closed":      suc.SiteClosed,
				}

				pt, err := influxClient.NewPoint(
					InfluxMeasurement,
					tags,
					fields,
					timeStamp,
				)
				if err != nil {
					fmt.Println(err)
					return
				}
				bp.AddPoint(pt)

				if verbose {
					fmt.Printf("%s (%d/%d available)\n", suc.Name, suc.AvailableStalls, suc.TotalStalls)
					fmt.Printf("Tags: %v\n", tags)
					fmt.Printf("Fields: %v\n", fields)
				}
			}

			// Write the batch
			err = dbClient.Write(bp)
			if err != nil {
				fmt.Println(err)
				return
			}

		}

		sleepTime := (15 + rand.Intn(30))
		if verbose {
			fmt.Printf("Sleeping for %d\n\n", sleepTime)
		}
		time.Sleep(time.Duration(sleepTime) * time.Second)

	}
}
