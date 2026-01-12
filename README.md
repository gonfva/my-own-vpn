# My Own VPN

This is a golang project to generate an app that helps manage VPN clients but stays in the notification area (both Windows and MacOS with https://github.com/getlantern/systray or similar)

The idea is that on connection time the application would create a VM (with options for EC2 in AWS or VM in Hetzner). Obviously there would also need to be some networking bits (VPC, subnet, IGW, SG, keypair in AWS). The VM would have WireGuard, and once the deployment is ready, it will setup a VPN connection with the VM.

When turning off, the VPN would disconnect and then the VM (and the rest of infra that has some cost) would be decommisioned.

The app has a section for storing the credentials. And also the posibility of chosing the region.
