![build](https://github.com/pgollangi/fastget/workflows/build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/pgollangi/fastget)](https://goreportcard.com/report/github.com/pgollangi/fastget)
![License: MIT](https://img.shields.io/github/license/pgollangi/fastget)

# FastGet

A CLI tool as well as go library to ultrafast download files over HTTP(s).

> DISCLAIMER: FastGet performance heavily reliant on the network and CPU performance of the client machine. More importantly HTTP(s) endpoint must allow partial requests presenting `Accept-Ranges` and accepting `Range` headers.