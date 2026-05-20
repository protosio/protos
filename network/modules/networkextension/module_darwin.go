//go:build darwin

package networkextension

import (
	"errors"
	"fmt"

	networkmodule "github.com/protosio/protos/internal/network/module"
)

var ErrUnavailable = errors.New("networkextension module requires a signed macOS system extension")

func New() (networkmodule.Module, error) {
	return nil, fmt.Errorf("networkextension module: %w", ErrUnavailable)
}
