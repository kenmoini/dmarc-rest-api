# README.md

## Status

[![GitHub release](https://img.shields.io/github/release/kenmoini/dmarc-rest-api.svg)](https://github.com/kenmoini/dmarc-rest-api/releases)
[![GitHub issues](https://img.shields.io/github/issues/kenmoini/dmarc-rest-api.svg)](https://github.com/kenmoini/dmarc-rest-api/issues)
[![Go Version](https://img.shields.io/badge/go-1.10-blue.svg)](https://golang.org/dl/)
[![Build Status](https://travis-ci.org/kenmoini/dmarc-rest-api.svg?branch=master)](https://travis-ci.org/kenmoini/dmarc-rest-api)
[![GoDoc](http://godoc.org/github.com/kenmoini/dmarc-rest-api?status.svg)](http://godoc.org/github.com/kenmoini/dmarc-rest-api)
[![SemVer](http://img.shields.io/SemVer/2.0.0.png)](https://semver.org/spec/v2.0.0.html)
[![License](https://img.shields.io/pypi/l/Django.svg)](https://opensource.org/licenses/BSD-2-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/kenmoini/dmarc-rest-api)](https://goreportcard.com/report/github.com/kenmoini/dmarc-rest-api)

## Installation

As with many Go utilities, a simple

    go get github.com/kenmoini/dmarc-rest-api

is enough to fetch, build and install.

## Dependencies

Aside from the standard library, this uses the following:

    go get -u github.com/intel/tfortools
    go get -u github.com/keltia/archive

## Usage - Single report via CLI

SYNOPSIS
```
dmarc-rest-api [-hvD] [--rest-server] <zipfile|xmlfile>

Example:

$ dmarc-rest-api /tmp/yahoo.com\!keltia.net\!1518912000\!1518998399.xml

Reporting by: Yahoo! Inc. — postmaster@dmarc.yahoo.com
From 2018-02-18 01:00:00 +0100 CET to 2018-02-19 00:59:59 +0100 CET

Domain: keltia.net
Policy: p=none; dkim=r; spf=r

Reports(1):
IP            Count   From       RFrom      RDKIM   RSPF
88.191.250.24 1       keltia.net keltia.net neutral pass
```

## Usage - As a REST API

SYNOPSIS
```
$ ./dmarc-rest-api --rest-server
```

This simple command will start the REST API Server listening on port 8080.  These are the following exposed endpoints:

- /api/v1/upload_bundle - The API endpoint accepting bundleFile input
- /healthz - A simple 200 OK Health Check for Kubernetes/OpenShift

From there, simply make a REST API call with the POST verb, as a *form-data* type submission, and with the DMARC bundle file passed via the body in a bundleFile input.

## Tests

Getting close to 80% coverage.  Need to add tests for REST API

## License

This is released under the BSD 2-Clause license.  See `LICENSE.md`.

## Contributing

I use Git Flow for this package so please use something similar or the usual github workflow.

1. Fork it ( https://github.com/kenmoini/dmarc-rest-api/fork )
2. Checkout the develop branch (`git checkout develop`)
3. Create your feature branch (`git checkout -b my-new-feature`)
4. Commit your changes (`git commit -am 'Add some feature'`)
5. Push to the branch (`git push origin my-new-feature`)
6. Create a new Pull Request
