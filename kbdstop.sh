#!/bin/sh

# Stop getty ONLY if we see the USB NFC reader connected
if ( lsusb | grep -q '16c0:27db' ) ; then 
        
        echo Stopping tty1 Getty 
        systemctl stop getty@tty1.service 

        # Disable console cursor
        echo 0 | tee /sys/class/graphics/fbcon/cursor_blink
else
        echo USB NFC Not found!
fi


true

