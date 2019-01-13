scimport

Tesla Supercharger data import utility.  Periodically invokes the
nearby-superchargers endpoint to retrieve Supercharger stall
utilization (free and total stalls) for the nearest Superchargers,
then dumps these measurements into an InfluxDB database.  One can then
use something like Grafana to visualize time-series measurements of
Supercharger occupancy.

This code has some issues because as of this writing, the community
version of InfluxDB is in flux, and it might be difficult to build the
required InfluxDB golang client library.

Use the -token flag to specify a Tesla authentication token (can be
created using the gettoken utility, or by any other means).

Use the -influx-url, -influx-database, and -influx-measurement flags
to specify where to write the data.  The defaults are to write a
measurement series named "chargers" in a database called "tesla" on a
local InfluxDB instance, with no authentication.  Note that
authentication and encryption are not supported at this time (it would
be fairly easy to add however).

Polling is every 30 seconds +/- 15 seconds (randomized to prevent
synchronization effects).
