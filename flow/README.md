# Implementation of IO, Stream, Fiber using go1.18 generics
![Coverage](https://img.shields.io/badge/Coverage-86.9%25-brightgreen)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/56db71f0cf6d4c76b796af26a1d7ef41)](https://app.codacy.com/gh/rprtr258/goflow?utm_source=github.com&utm_medium=referral&utm_content=rprtr258/goflow&utm_campaign=Badge_Grade_Settings)
[![Go Reference](https://pkg.go.dev/badge/github.com/rprtr258/goflow.svg)](https://pkg.go.dev/github.com/rprtr258/goflow)
[![GoDoc](https://godoc.org/github.com/rprtr258/goflow?status.svg)](https://godoc.org/github.com/rprtr258/goflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/rprtr258/goflow)](https://goreportcard.com/report/github.com/rprtr258/goflow)
[![Version Badge](https://img.shields.io/github/v/tag/rprtr258/goflow)](https://img.shields.io/github/v/tag/rprtr258/goflow)
![Go](https://github.com/rprtr258/goflow/workflows/Go/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/rprtr258/goflow/branch/master/graph/badge.svg?token=WXVKKB4EWO)](https://codecov.io/gh/rprtr258/goflow)

This library is an attempt to fill the gap of a decent generics streaming libraries in Go lang. The existing alternatives do not yet use Go 1.18 generics to their full potential.

The design is inspired by awesome Scala libraries [cats-effect](https://typelevel.org/cats-effect/) and [fs2](https://fs2.io/).
