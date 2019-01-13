gotesla
=======

Tesla API client library and utilities written in golang.

Tesla, Inc. has created an externally-accessible API for query and (to
a very limited extent) controlling their vehicles, such as the Model
S, Model X, and Model 3.  This API is used by their mobile
applications on iOS and Android.  It is totally undocumented and
unsupported by Tesla, however a number of third-party applications
(desktop, mobile, Web sites) have been created that use this API.

This repository contains a limited implementation of a
[golang](https://golang.org) client library to access that API, along
with a couple of utilities that exercise the client library and API.
It was created solely for the author's personal use on a specific
project (the aforementioned utilities), and is being made available in
the hopes that it is useful to others.

Crowd-sourced, reverse-engineered information on the Tesla API has
been obtained from (https://www.teslaapi.io/) and related sources.
The author acknowledges and greatly appreciates the efforts of those
who have contributed to this effort.

gettoken
--------

A utility to obtain an authentication token from Tesla, used for
various API calls.  Requires a valid MyTesla account (email address)
and password.

scimport
--------

Imports Supercharger stall utilization data to an InfluxDB timeseries
database.

Copyright
---------

Copyright (C) 2019 Bruce A. Mah.  All rights reserved.  Distributed
under a BSD-style license, see the LICENSE file for more information.

