# rv-homekit

rv-homekit is a small GO app used to act as a proxy/adapter between the OneControl interface from LCI and Apple's HomeKit

It is designed to be able to run on a raspberry pi plugged into the ethernet port on the OneControl gateway and joined to 
your own WiFi network if they are separate.

## Installation

First you need to prepare a RaspberryPi as your host for the application. The OS that I'm using myself is [Ubuntu 20.04 server](https://ubuntu.com/download/raspberry-pi).

If you have a separate WiFi network that you use for your devices configure the wireless on the pi to connect to that network

## Configuration

There are a few settings that need to be configured and saved in a config.json file in the same directory as the application.

The config file needs the following information:
* Bridge Name
* Bridge Pin
* OpenHAB host. (Should be http://192.168.1.4:8080)

Here's an example config.

```json
{
  "bridgeName": "Big Blue",
  "openHabServer": "http://192.168.1.4:8080",
  "pin": "00102003"
}
```


