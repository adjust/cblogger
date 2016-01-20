#!/bin/bash

WORKDIR=`pwd`

### update system
yum -y update

### install go
yum -y install golang

### install nginx
yum -y install nginx
cp nginx/nginx.conf /etc/nginx
cp nginx/cblogger.conf /etc/nginx/conf.d
/etc/init.d/nginx start
/sbin/chkconfig nginx on

### install daemonize
wget https://github.com/bmc/daemonize/archive/release-1.7.6.tar.gz
tar xf release-1.7.6.tar.gz
cd ${WORKDIR}/daemonize-release-1.7.6
./configure --prefix=/usr
make
make install
cd ${WORKDIR}

rm -rf daemonize-release-1.7.6
rm release-1.7.6.tar.gz

### build binary
mkdir -p /usr/local/bin
go build -o /usr/local/bin/cblogger
RETURNCODE=$?
if [[ $RETURNCODE != 0 ]]
then
    echo "Building binary failed"
    exit $RETURNCODE
fi

### copy init script
cp init/cblogger /etc/init.d
/etc/init.d/cblogger start
/sbin/chkconfig cblogger on
