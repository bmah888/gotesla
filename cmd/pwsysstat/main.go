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
	"math/rand"
	"net/http"
	"time"
)

var hostname string
var email string
var password string

func main() {
	var verbose bool

	// Seed random number generator, for semi-random polling interval
	rand.Seed(time.Now().UTC().UnixNano())

	// Command-line arguments
	flag.StringVar(&hostname, "hostname", "teg", "Powerwall gateway hostname")
	flag.StringVar(&email, "email", "", "Email address for login")
	flag.StringVar(&password, "password", "", "Password for login")
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
	if (email != "" && password != "") {
		pwa, err = gotesla.GetPowerwallAuth(client, hostname, email, password)
		if err != nil {
			log.Fatalf("PowerwallAuth: %v\n", err);
		}
	}

	// Maybe print out some stuff from the token
	if (verbose) {
		if pwa != nil {
			fmt.Printf("email %s\n", pwa.Email)
			fmt.Printf("token %s\n", pwa.Token)
			fmt.Printf("timestamp %s\n", pwa.Timestamp.Format(time.UnixDate))
		}
	}

	sysstat, err := gotesla.GetSystemStatus(client, hostname, pwa)
	if err != nil {
		log.Printf("GetSystemStatus: %v\n", err)
		return
	}

	fmt.Printf("Batteries: %d\n", sysstat.AvailableBlocks)
	fmt.Printf("SystemIslandState: %s\n", sysstat.SystemIslandState)
	fmt.Printf("System target power: %f\n", sysstat.BatteryTargetPower)
	fmt.Printf("System nominal full pack energy: %d\n", sysstat.NominalFullPackEnergy)
	fmt.Printf("System nominal energy remaining: %d\n", sysstat.NominalEnergyRemaining)
	fmt.Printf("System computed SOE: %d%%\n",
		sysstat.NominalEnergyRemaining * 100 /
		sysstat.NominalFullPackEnergy)

	fmt.Printf("\n")
	fmt.Printf("%3s %16s %16s %8s %10s %10s %10s\n", "#", "Part Number", "Serial Number", "Full", "Remaining", "Charged", "Discharged")
	
	var i int
	var totalCharged, totalDischarged int
	for i = 0; i < sysstat.AvailableBlocks; i++ {
		fmt.Printf("%3d %16s %16s %8d %10d %10d %10d\n",
			i,
			sysstat.BatteryBlocks[i].PackagePartNumber,
			sysstat.BatteryBlocks[i].PackageSerialNumber,
			sysstat.BatteryBlocks[i].NominalFullPackEnergy,
			sysstat.BatteryBlocks[i].NominalEnergyRemaining,
			sysstat.BatteryBlocks[i].EnergyCharged,
			sysstat.BatteryBlocks[i].EnergyDischarged)
		totalCharged += sysstat.BatteryBlocks[i].EnergyCharged
		totalDischarged += sysstat.BatteryBlocks[i].EnergyDischarged
	}
	fmt.Printf("%3s %16s %16s %8d %10d %10d %10d\n", "SYS", "", "", sysstat.NominalFullPackEnergy, sysstat.NominalEnergyRemaining, totalCharged, totalDischarged)
}
