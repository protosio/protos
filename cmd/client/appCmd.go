package main

import (
	"context"
	"fmt"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
)

func getApps(client pbApic.ProtosClientApiClient) error {
	log.Debug("Retrieving apps")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apps, err := client.GetApps(ctx, &pbApic.GetAppsRequest{})
	if err != nil {
		return err
	}
	fmt.Println(apps)
	return nil
}
