pwimport
========

Tesla Powerwall data import utility.  Periodically queries a Tesla
Powerwall gateway via the local LAN and pulls statistics into a
InfluxDB database.  Specifically, it pulls the aggregate meters, state
of energy, and grid status.  These can be visualized (for example) in
a Grafana dashboard.

Note that no authentication parameters are required for the Powerwall
APIs used by this utility.

Use the `-influx-url`, `-influx-database`, and `-influx-measurement`
flags to specify where to write the data.  The defaults are to write a
measurement series named `powerwall` in a database called `tesla` on a
local InfluxDB instance, with no authentication.  Note that
authentication and encryption are not supported at this time (it would
be fairly easy to add however).

Use the `-hostname` flag to specify the hostname of the Powerwall
gateway to query (the default is `teg`).  Note the gateway must be
reachable from the host running this program; it will not work if the
Powerwall is relying solely on cellular data for connectivity to
Tesla's backend servers.

Use the `-poll` flag to specify the time to pause between polling
cycles, in seconds.  The default is 10 seconds.  Polling cycles as low
as 5 seconds have been run with no ill effects, although as a matter
of common sense it is probably wise to avoid polling the gateway too
frequently.

