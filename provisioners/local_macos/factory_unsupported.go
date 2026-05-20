//go:build !darwin

package localmacos

import "github.com/protosio/protos/internal/provisioners"

func NewFactory() provisioners.ProvisionerFactory {
	return nil
}
