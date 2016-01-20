#!/bin/bash

### build binary
mkdir -p /usr/local/bin
go build -o /usr/local/bin/cblogger
RETURNCODE=$?
if [[ $RETURNCODE != 0 ]]
then
    echo "Building binary failed"
    exit $RETURNCODE
fi

sudo /etc/init.d/cblogger restart
