//go:build tools
// +build tools

package main

import (
	_ "github.com/vektra/mockery/v2"
)

//go:generate go run github.com/vektra/mockery/v2
