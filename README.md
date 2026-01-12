# My Own VPN

This is a golang project to generate an app that helps manage VPN clients but stays in the notification area (both Windows and MacOS with https://github.com/getlantern/systray or similar)

The idea is that on connection time the application would create a VM (with options for EC2 in AWS or VM in Hetzner), and when turning off, the VM would be decommisioned.

The app has a section for storing the credentials.
