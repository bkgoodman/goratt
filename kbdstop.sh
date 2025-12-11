#!/bin/sh

echo Getty Stop
if ( lsusb | grep -q '16c0:27db' ) ; then echo Stopping tty1 Getty ; systemctl stop getty@tty1.service ; fi

true

