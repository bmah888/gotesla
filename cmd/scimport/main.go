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
	"github.com/bmah888/gotesla"
	"log"
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

func main() {
	var verbose = false

	// Seed random number generator, for semi-random polling interval
	rand.Seed(time.Now().UTC().UnixNano())

	// Command-line arguments
	flag.StringVar(&InfluxURL, "influx-url", "http://localhost:8086",
		"Influx database server URL")
	flag.StringVar(&InfluxDb, "influx-database", "tesla",
		"Influx database name")
	flag.StringVar(&InfluxMeasurement, "influx-measurement", "chargers",
		"Influx measurement name")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	flag.StringVar(&(gotesla.TokenCachePath), "token-cache", gotesla.TokenCachePath, "Path to Telsa token cache file")

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
		log.Fatalf("GetVehicles: %v\n", err)
		return
	}

	if verbose {
		fmt.Printf("%d vehicles retrieved\n", len(*vehicles))
	}

	// Get a new HTTP client for InfluxDB
	dbClient, err := influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr: InfluxURL,
	})
	if err != nil {
		log.Fatalf("NewHTTPClient: %v\n", err)
	}
	defer dbClient.Close()

	for {

		// See if token will be expiring soon (less than a day)
		h := gotesla.TokenLifetime(token).Hours()
		if verbose {
			fmt.Printf("Token lifetime (hours): %f\n", h)
		}
		if h < 24.0 {
			// Expiring soon, attempt to refresh it.
			token2, err := gotesla.RefreshAndCacheToken(client, token)
			if err != nil {
				log.Printf("RefreshAndCacheToken: %v\n", err)
			} else {
				token = token2
				if verbose {
					log.Printf("Refresh token successful\n")
				}
			}
		}

		for _, v := range *vehicles {
			if verbose {
				fmt.Printf("Vehicle: id %s VIN %s\n", v.IDS, v.Vin)
			}

			nc, err := gotesla.GetNearbyChargers(client, token, v.IDS )
			if err != nil {
				log.Printf("GetNearbyChargers: %v\n", err)
				continue
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
				Precision: "s",
			})
			if err != nil {
				log.Printf("NewBatchPoints: %v\n", err)
				continue
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
					log.Printf("NewPoint: %v\n", err)
					continue
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
				log.Printf("Write: %v\n", err)
				continue
			}

		}

		sleepTime := (15 + rand.Intn(30))
		if verbose {
			fmt.Printf("Sleeping for %d\n\n", sleepTime)
		}
		time.Sleep(time.Duration(sleepTime) * time.Second)

	}
}
