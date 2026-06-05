//go:build darwin

package control

import (
	"context"

	hostagentclient "github.com/protosio/protos/internal/hostagent/client"
)

func Stop(ctx context.Context) error {
	client, err := hostagentclient.New()
	if err != nil {
		return err
	}
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		done <- client.Shutdown()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
