# Nym

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/nymtech/nym/blob/master/LICENSE)
<!-- [![Build Status](https://travis-ci.com/jstuczyn/CoconutGo.svg?branch=master)](https://travis-ci.com/jstuczyn/CoconutGo)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/0xacab.org/jstuczyn/CoconutGo)
[![Coverage Status](http://codecov.io/github/jstuczyn/CoconutGo/coverage.svg?branch=master)](http://codecov.io/github/jstuczyn/CoconutGo?branch=master) -->

This is the Nym core platform. It includes a Go implementation of the [Coconut](https://arxiv.org/pdf/1802.07344.pdf) selective disclosure credentials scheme. Coconut supports threshold issuance on multiple public and private attributes, re-randomization and multiple unlinkable selective attribute revelations.

The implementation is based on the [existing Python version](https://github.com/asonnino/coconut).

Nym uses a [Tendermint](https://tendermint.com/) blockchain to keep track of credential issuance and prevent double-spending of credentials, and contains server-side to support querying of credentials. There is client-side code to re-randomize credentials.

For more information, see the [documentation](https://github.com/nymtech/docs/tree/master/content).
