pwsystat
========

Tesla Powerwall system_status query utility.  This API returns a JSON
structure that prints (among other things) the capacity of each
battery in a Powerwall system, as well as the system as a whole; the
utility prints this information in a human-readable form.  That
information is useful for tracking battery degradation over time.

Use the `-hostname` flag to specify the hostname of the Powerwall
gateway to query (the default is `teg`).  Note the gateway must be
reachable from the host running this program; it will not work if the
Powerwall is relying solely on cellular data for connectivity to
Tesla's backend servers.

Use the `-email` and `-password` options for authentication.

