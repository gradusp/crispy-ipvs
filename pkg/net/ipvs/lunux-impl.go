//go:build linux
// +build linux

package ipvs

import (
	"context"
)

//NewAdmin manes inst of Ipvs.Admin
func NewAdmin(_ context.Context) Admin {
	return nil
}
