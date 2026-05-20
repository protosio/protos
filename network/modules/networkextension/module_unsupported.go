//go:build !darwin

package networkextension

import (
	"fmt"
	"runtime"

	networkmodule "github.com/protosio/protos/internal/network/module"
)

func New() (networkmodule.Module, error) {
	return nil, fmt.Errorf("networkextension module is not available for %s", runtime.GOOS)
}
