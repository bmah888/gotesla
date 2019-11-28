Docker Container Builds
=======================

This directory contains Dockerfiles for creating several types of
containers for use with gotesla programs.  The intended usage is to
create long-running containers for data-gathering programs.

1.  `Dockerfile.pwimport` creates a Docker container that runs an
instance of `pwimport`.  Build it thusly from the top-level source
directory:

----
# docker build -f build/package/Dockerfile.pwimport -t bmah888/gotesla/cmd/pwimport:latest .
----

The following environment variables map to arguments to pwimport:

`PWI_HOST` : `-hostname`
`PWI_INFLUX_DB` : `-influx-database`
`PWI_INFLUX_MEASUREMENT` : `-influx-measurement`
`PWI_INFLUX_URL` : `-influx-url`
`PWI_POLL` : `-poll`

An example invocation of this container:

----
# docker run --env PWI_INFLUX_URL="http://influxdb:8086" --env
PWI_POLL=5 --network opt_default --name pwimport --detach -t bmah888/gotesla/cmd/pwimport:latest
----

2.  `Dockerfile.scimport` creates a Dockerfile container to run an
instance of `scimport`.


