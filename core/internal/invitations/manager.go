package invitations

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const ChannelMDNS = "mdns"

type Invite struct {
	InviteID         string
	Channel          string
	OrganisationID   string
	OrganisationName string
	DeviceName       string
	PeerID           string
	PublicKey        string
	VerificationCode string
	VerificationHash string
	Port             int
	P2PAddrs         []string
	SwarmionAddrs    []string
	ExpiresAt        time.Time
	AdvertiseName    string
	AdvertiseService string
}

type NearbyInvite struct {
	InviteID         string
	Channel          string
	OrganisationID   string
	OrganisationName string
	DeviceName       string
	PeerID           string
	PublicKey        string
	VerificationHash string
	P2PAddrs         []string
	SwarmionAddrs    []string
	ExpiresAt        time.Time
	HostName         string
	Port             int
	IPs              []string
}

type Channel interface {
	Name() string
	StartInvite(context.Context, Invite) (Invite, error)
	Browse(context.Context, time.Duration) ([]NearbyInvite, error)
	Stop()
}

type Manager struct {
	mu       sync.Mutex
	channels map[string]Channel
	nearby   map[string]NearbyInvite
}

func NewManager(channels ...Channel) (*Manager, error) {
	manager := &Manager{
		channels: map[string]Channel{},
		nearby:   map[string]NearbyInvite{},
	}
	for _, channel := range channels {
		if err := manager.Register(channel); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (m *Manager) Register(channel Channel) error {
	if channel == nil {
		return errors.New("invite channel is nil")
	}
	name := normalizeChannel(channel.Name())
	if name == "" {
		return errors.New("invite channel name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.channels[name]; exists {
		return fmt.Errorf("invite channel %q is already registered", name)
	}
	m.channels[name] = channel
	return nil
}

func (m *Manager) StartInvite(ctx context.Context, channelName string, invite Invite) (Invite, error) {
	channel, err := m.channel(channelName)
	if err != nil {
		return Invite{}, err
	}
	invite.Channel = channel.Name()
	started, err := channel.StartInvite(ctx, invite)
	if err != nil {
		return Invite{}, err
	}
	if strings.TrimSpace(started.Channel) == "" {
		started.Channel = channel.Name()
	}
	return started, nil
}

func (m *Manager) Browse(ctx context.Context, channelName string, timeout time.Duration) ([]NearbyInvite, error) {
	channels, err := m.selectedChannels(channelName)
	if err != nil {
		return nil, err
	}
	nearby := map[string]NearbyInvite{}
	var browseErrs []error
	for _, channel := range channels {
		items, err := channel.Browse(ctx, timeout)
		if err != nil {
			browseErrs = append(browseErrs, fmt.Errorf("%s: %w", channel.Name(), err))
			continue
		}
		for _, item := range items {
			if strings.TrimSpace(item.Channel) == "" {
				item.Channel = channel.Name()
			}
			if item.ExpiresAt.Before(time.Now()) {
				continue
			}
			nearby[item.cacheKey()] = item
		}
	}
	if len(nearby) == 0 && len(browseErrs) > 0 {
		return nil, errors.Join(browseErrs...)
	}

	out := make([]NearbyInvite, 0, len(nearby))
	for _, item := range nearby {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OrganisationName != out[j].OrganisationName {
			return out[i].OrganisationName < out[j].OrganisationName
		}
		if out[i].DeviceName != out[j].DeviceName {
			return out[i].DeviceName < out[j].DeviceName
		}
		if out[i].Channel != out[j].Channel {
			return out[i].Channel < out[j].Channel
		}
		return out[i].PeerID < out[j].PeerID
	})

	m.mu.Lock()
	if m.nearby == nil {
		m.nearby = map[string]NearbyInvite{}
	}
	for _, item := range out {
		m.nearby[item.cacheKey()] = item
	}
	m.mu.Unlock()

	return out, nil
}

func (m *Manager) Find(ctx context.Context, channelName string, organisationID string, peerID string, inviteID string) (NearbyInvite, error) {
	organisationID = strings.TrimSpace(organisationID)
	peerID = strings.TrimSpace(peerID)
	inviteID = strings.TrimSpace(inviteID)
	if organisationID == "" {
		return NearbyInvite{}, fmt.Errorf("organisation id is required")
	}
	if peerID == "" {
		return NearbyInvite{}, fmt.Errorf("peer id is required")
	}

	if item, ok := m.findCached(channelName, organisationID, peerID, inviteID); ok {
		return item, nil
	}
	if _, err := m.Browse(ctx, channelName, 3*time.Second); err != nil {
		return NearbyInvite{}, err
	}
	if item, ok := m.findCached(channelName, organisationID, peerID, inviteID); ok {
		return item, nil
	}
	return NearbyInvite{}, fmt.Errorf("nearby organisation %s from peer %s was not found", organisationID, peerID)
}

func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	channels := make([]Channel, 0, len(m.channels))
	for _, channel := range m.channels {
		channels = append(channels, channel)
	}
	m.mu.Unlock()
	for _, channel := range channels {
		channel.Stop()
	}
}

func (m *Manager) channel(channelName string) (Channel, error) {
	if m == nil {
		return nil, fmt.Errorf("invite manager is nil")
	}
	name := normalizeChannel(channelName)
	if name == "" {
		name = ChannelMDNS
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	channel := m.channels[name]
	if channel == nil {
		return nil, fmt.Errorf("invite channel %q is not available", name)
	}
	return channel, nil
}

func (m *Manager) selectedChannels(channelName string) ([]Channel, error) {
	if m == nil {
		return nil, fmt.Errorf("invite manager is nil")
	}
	name := normalizeChannel(channelName)
	m.mu.Lock()
	defer m.mu.Unlock()
	if name != "" {
		channel := m.channels[name]
		if channel == nil {
			return nil, fmt.Errorf("invite channel %q is not available", name)
		}
		return []Channel{channel}, nil
	}
	channels := make([]Channel, 0, len(m.channels))
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		channels = append(channels, m.channels[name])
	}
	if len(channels) == 0 {
		return nil, fmt.Errorf("no invite channels are available")
	}
	return channels, nil
}

func (m *Manager) findCached(channelName string, organisationID string, peerID string, inviteID string) (NearbyInvite, bool) {
	if m == nil {
		return NearbyInvite{}, false
	}
	channelName = normalizeChannel(channelName)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range m.nearby {
		if channelName != "" && item.Channel != channelName {
			continue
		}
		if item.OrganisationID != organisationID || item.PeerID != peerID {
			continue
		}
		if inviteID != "" && item.InviteID != inviteID {
			continue
		}
		if item.ExpiresAt.Before(time.Now()) {
			delete(m.nearby, item.cacheKey())
			continue
		}
		return item, true
	}
	return NearbyInvite{}, false
}

func normalizeChannel(channelName string) string {
	return strings.ToLower(strings.TrimSpace(channelName))
}

func (item NearbyInvite) cacheKey() string {
	return item.Channel + "/" + item.OrganisationID + "/" + item.PeerID + "/" + item.InviteID
}
