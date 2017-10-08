# Network Down -> reboot -n Hook
## By Andrew Lee

A simple program to throw into rc.local (mainly for Pi's that need to be constantly running stateless application [do not use if rebooting your Pi will cause data loss] with an unstable network) that will restart the computer anytime there is no value for wlan0
