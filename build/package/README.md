Docker Container Builds
=======================

This directory contains Dockerfiles for creating several types of
containers for use with gotesla programs.  The intended usage is to
create long-running containers for data-gathering programs.

`Dockerfile.pwimport` creates a Docker container that runs an
instance of `pwimport`.  Build it thusly from the top-level source
directory:

    # docker build -f build/package/Dockerfile.pwimport -t bmah888/gotesla/cmd/pwimport:latest .

The following environment variables map to arguments to pwimport:

    PWI_HOST : -hostname
    PWI_INFLUX_DB : -influx-database
    PWI_INFLUX_MEASUREMENT : -influx-measurement
    PWI_INFLUX_URL : -influx-url
    PWI_POLL : -poll

An example invocation of this container:

    # docker run --env PWI_INFLUX_URL="http://influxdb:8086" --env PWI_POLL=5 --network opt_default --name pwimport --detach -t bmah888/gotesla/cmd/pwimport:latest

`Dockerfile.scimport` creates a Dockerfile container to run an
instance of `scimport`.  A sample build of this container can be done
like this:

    # docker build -f build/package/Dockerfile.scimport -t bmah888/gotesla/cmd/scimport:latest .

The following environment variables map to arguments to pwimport:

    SCI_HOST : -hostname
    SCI_INFLUX_DB : -influx-database
    SCI_INFLUX_MEASUREMENT : -influx-measurement
    SCI_INFLUX_URL : -influx-url
    SCI_TOKEN_CACHE : -token-cache

To use this container, pre-fetch a Tesla authentication token using
the gettoken utility, giving a `-token-cache` flag pointing to some
suitable location. It's probably best if this token cache is not
shared by any other programs, as scimport will refresh the token as
its lifetime approaches.  Of course the token cache file should also
not be readable by unauthorized parties, because that gives access to
all of the remote Tesla features.

In this example, we create a new directory to hold the token cache
file.  We'll map that directory into the container.  The reason for
this is that scimport will automatically try to renew the stored token
as it approaches its expiration time, but in order to do that, it
needs a directory to store the new token.

    % mkdir ${PWD}/token
    % gettoken --email user@example.com --password secret-password --token-cache ${PWD}/token/token-cache.json

Map the directory containing the token-cache file into the container
in the container's run command, like this.

    # docker run --env SCI_INFLUX_URL="http://influxdb:8086" --env SCI_TOKEN_CACHE="/token/token-cache.json" --mount type=bind,source="${PWD}/token,target=/token" --network opt_default --name scimport --detach -t bmah888/gotesla/cmd/scimport:latest

In this same directory is a `docker-compose.yml` file, which gives an
example of how to run `pwimport` and `scimport` in their containers
alongside instances of InfluxDB and Grafana.

