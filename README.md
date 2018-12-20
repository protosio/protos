# Protos - own your digital identity and data by self-hosting

<p align="center">
<img src="https://protos.io/static/img/protos_logo.png"  width="320" height="117" alt="Protos logo" title="Protos logo">
</p>

**Protos** is an open-source project that enables individuals and small organizations to take full control of their digital identity and data, by allowing them to self-host applications on public cloud providers or their own hardware. Currently Protos is under heavy development and only has alpha quality releases.

## Features ##

Some of the following features are not fully implemented yet but this project aims to deliver at the minimum the following features:

- self-sovereign identity - currently a user is required to own a domain name that can be used with a Protos instance. In the future a harder form of cryptographic identity will be implemented based on public-key cryptography, leveraging the framework developed by [DIF](https://identity.foundation).
- application store - installing an application is as easy as clicking a button.
- service based architecture - applications can leverage resources and services provided by other applications installed on the platform.
- full data ownership - applications store their data locally on the Protos instance. Local data is encrypted at rest while backups sent to 3rd party services are also encrypted.
- easy migration - the whole instances together with its data, can be easily migrated to a different hosting provider. DNS records are switched automatically and users can continue using their applications.

## Screenshot ##

<p align="center">
<img src="https://protos.io/static/img/screenshot.png"  width="640" height="441" alt="screenshot" title="screenshot">
</p>

## Dependencies ##

Protos leverages [docker-engine](https://docs.docker.com/install/) to run applications, and requires version `v18.03` or higher.

## Running ##

- download the latest release from https://releases.protos.io
```
sudo wget https://releases.protos.io/0.0.1-alpha.1/protos -O /usr/local/bin/protos
```
- mark the Protos binary executable
```
sudo chmod +x /usr/local/bin/protos
```
- copy and customize a configuration based on [the one](https://github.com/protosio/protos/blob/master/protos.yaml) in this repository
- create a data directory for Protos
```
sudo mkdir /opt/protos
```
- run Protos in init mode and complete the setup process
```
sudo /usr/local/bin/protos --config /etc/protos.yml --loglevel debug init
```

## Developing ##

