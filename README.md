# NIBE

This is a Go module and CLI for interacting with the NIBE S-series heat pumps using the Local REST API. This API can be enabled in firmwares starting in fall 2025.

Enable and configure the local API in menu: 7.5.15

## Usage

You need to provide the certificate fingerprint and device serial number to be able to correctly authenticate the heat pump.

* Username: from menu 7.5.15.
* Password: from menu 7.5.15.
* Fingerprint: from menu 7.5.15.
* Serial number: from menu 1.1.1.
* Endpoint: `https://IP-or-hostname:8443`.

## API documentation

You can view the API documentation of the pump by opening up the endpoint address in your browser.
