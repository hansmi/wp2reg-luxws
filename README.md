# wp2reg-luxws

[![Latest release](https://img.shields.io/github/v/release/hansmi/wp2reg-luxws)][releases]
[![Release workflow](https://github.com/hansmi/wp2reg-luxws/actions/workflows/release.yaml/badge.svg)](https://github.com/hansmi/wp2reg-luxws/actions/workflows/release.yaml)
[![CI workflow](https://github.com/hansmi/wp2reg-luxws/actions/workflows/ci.yaml/badge.svg)](https://github.com/hansmi/wp2reg-luxws/actions/workflows/ci.yaml)
[![Go reference](https://pkg.go.dev/badge/github.com/hansmi/wp2reg-luxws.svg)](https://pkg.go.dev/github.com/hansmi/wp2reg-luxws)

A collection of [Go][golang] packages for working with the `Lux_WS` protocol
used for remote control in Luxtronik 2.x heat pump controllers manufactured
and/or deployed by the following companies:

* Alpha Innotec
* NIBE
* Novelan
* possibly other companies and/or brands

The websocket-based protocol was introduced in firmware version 3.81. The code
was developed and tested using wp2reg version 3.85.6.

## Prometheus exporter

The primary purpose of this code is to export all informational values for
consumption by Prometheus. See the [`luxws-exporter`](./luxws-exporter)
directory for details.

## Installation

Pre-built binaries are provided for all [releases][releases]:

* Binary archives for Linux, Windows and Mac OS (`.tar.gz`, `.zip`)
* Debian/Ubuntu (`.deb`)
* RHEL/Fedora (`.rpm`)

With the source being available it's also possible to produce custom builds
directly using [Go][golang] or [GoReleaser][goreleaser].

[golang]: https://golang.org/
[websocket]: https://en.wikipedia.org/wiki/WebSocket
[releases]: https://github.com/hansmi/wp2reg-luxws/releases/latest
[goreleaser]: https://goreleaser.com/
