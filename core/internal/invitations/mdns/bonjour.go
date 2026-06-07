package mdns

import (
	"context"
	"time"

	"github.com/protosio/protos/internal/invitations"
)

type bonjourAdvertisement interface {
	Shutdown()
}

func filterNearby(items []invitations.NearbyInvite) []invitations.NearbyInvite {
	nearby := map[string]invitations.NearbyInvite{}
	for _, item := range items {
		if item.ExpiresAt.Before(time.Now()) {
			continue
		}
		key := cacheKey(item)
		if existing, found := nearby[key]; found && !item.ExpiresAt.After(existing.ExpiresAt) {
			continue
		}
		nearby[key] = item
	}
	out := make([]invitations.NearbyInvite, 0, len(nearby))
	for _, item := range nearby {
		out = append(out, item)
	}
	return out
}

func browseBonjourWithTimeout(ctx context.Context, timeout time.Duration, browse func(context.Context) ([]invitations.NearbyInvite, error)) ([]invitations.NearbyInvite, error) {
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	items, err := browse(browseCtx)
	if err != nil {
		return nil, err
	}
	return filterNearby(items), nil
}
