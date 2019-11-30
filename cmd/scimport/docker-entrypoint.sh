#! /bin/bash
set -e

ARGS=""

if [ "x$SCI_INFLUX_DB" != 'x' ]; then
	ARGS="$ARGS -influx-database $SCI_INFLUX_DB"
fi

if [ "x$SCI_INFLUX_MEASUREMENT" != 'x' ]; then
	ARGS="$ARGS -influx-measurement $SCI_INFLUX_MEASUREMENT"
fi

if [ "x$SCI_INFLUX_URL" != 'x' ]; then
	ARGS="$ARGS -influx-url $SCI_INFLUX_URL"
fi

if [ "x$SCI_POLL" != 'x' ]; then
	ARGS="$ARGS -poll $SCI_POLL"
fi

if [ "x$SCI_TOKEN_CACHE" != 'x' ]; then
	ARGS="$ARGS -token-cache $SCI_TOKEN_CACHE"
fi

/scimport $ARGS


