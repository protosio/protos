package mdns

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"

	"github.com/protosio/protos/internal/invitations"
	"github.com/protosio/protos/internal/util"
)

const (
	ServiceTCP    = "_protos._tcp"
	ServiceDomain = "local."

	defaultInviteTTL = 10 * time.Minute
	txtVersion       = "1"
)

var log = util.GetLogger("inviteMDNS")

type Channel struct {
	mu           sync.Mutex
	advertise    bonjourAdvertisement
	inviteCancel context.CancelFunc
	current      invitations.Invite
}

func NewChannel() *Channel {
	return &Channel{}
}

func (c *Channel) Name() string {
	return invitations.ChannelMDNS
}

func (c *Channel) StartInvite(_ context.Context, invite invitations.Invite) (invitations.Invite, error) {
	if c == nil {
		return invitations.Invite{}, fmt.Errorf("mDNS invite channel is nil")
	}
	if strings.TrimSpace(invite.OrganisationID) == "" {
		return invitations.Invite{}, fmt.Errorf("organisation id is required")
	}
	if strings.TrimSpace(invite.OrganisationName) == "" {
		return invitations.Invite{}, fmt.Errorf("organisation name is required")
	}
	if strings.TrimSpace(invite.PeerID) == "" {
		return invitations.Invite{}, fmt.Errorf("peer id is required")
	}
	if invite.Port <= 0 {
		return invitations.Invite{}, fmt.Errorf("p2p port is required")
	}
	if len(invite.SwarmionAddrs) == 0 {
		return invitations.Invite{}, fmt.Errorf("at least one Swarmion bootstrap address is required")
	}
	invite.JoinMode = invitations.NormalizeInviteJoinMode(invite.JoinMode)
	if !invitations.ValidInviteJoinMode(invite.JoinMode) {
		return invitations.Invite{}, fmt.Errorf("invalid invite join mode %q", invite.JoinMode)
	}
	if strings.TrimSpace(invite.InviteID) == "" {
		invite.InviteID = xid.New().String()
	}
	invite.Channel = c.Name()
	if invite.ExpiresAt.IsZero() {
		invite.ExpiresAt = time.Now().Add(defaultInviteTTL)
	}
	invite.P2PAddrs = dedupeStrings(invite.P2PAddrs)
	invite.SwarmionAddrs = dedupeStrings(invite.SwarmionAddrs)
	if strings.TrimSpace(invite.VerificationCode) == "" {
		code, err := invitations.GenerateVerificationCode()
		if err != nil {
			return invitations.Invite{}, err
		}
		invite.VerificationCode = code
	}
	invite.VerificationHash = invitations.InviteVerificationHash(invite, invite.VerificationCode)
	if strings.TrimSpace(invite.AdvertiseName) == "" {
		invite.AdvertiseName = invite.OrganisationName
	}
	invite.AdvertiseService = ServiceTCP

	instanceName := inviteInstanceName(invite)
	txt := inviteTXT(invite)
	log.Infof(
		"publishing mDNS invite %q instance=%q service=%q domain=%q port=%d txt_records=%d swarmion_addrs=%d",
		invite.InviteID,
		instanceName,
		ServiceTCP,
		ServiceDomain,
		invite.Port,
		len(txt),
		len(invite.SwarmionAddrs),
	)
	advertise, err := startBonjourAdvertisement(instanceName, ServiceTCP, ServiceDomain, invite.Port, txt)
	if err != nil {
		return invitations.Invite{}, fmt.Errorf("start Bonjour advertisement: %w", err)
	}
	log.Infof("published mDNS invite %q instance=%q", invite.InviteID, instanceName)

	inviteCtx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.stopLocked()
	c.advertise = advertise
	c.inviteCancel = cancel
	c.current = invite
	c.mu.Unlock()

	go c.stopInviteWhenExpired(inviteCtx, invite.InviteID, time.Until(invite.ExpiresAt))
	return invite, nil
}

func (c *Channel) Stop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *Channel) Browse(ctx context.Context, timeout time.Duration) ([]invitations.NearbyInvite, error) {
	if c == nil {
		return nil, fmt.Errorf("mDNS invite channel is nil")
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	out, err := browseBonjour(ctx, timeout)
	if err != nil {
		return nil, err
	}
	log.Infof("mDNS invite browse completed with %d candidate(s)", len(out))
	sort.Slice(out, func(i, j int) bool {
		if out[i].OrganisationName != out[j].OrganisationName {
			return out[i].OrganisationName < out[j].OrganisationName
		}
		if out[i].DeviceName != out[j].DeviceName {
			return out[i].DeviceName < out[j].DeviceName
		}
		return out[i].PeerID < out[j].PeerID
	})

	return out, nil
}

func (c *Channel) stopInviteWhenExpired(ctx context.Context, inviteID string, delay time.Duration) {
	if delay <= 0 {
		delay = time.Nanosecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
		c.mu.Lock()
		if c.current.InviteID == inviteID {
			c.stopLocked()
		}
		c.mu.Unlock()
	}
}

func (c *Channel) stopLocked() {
	if c.inviteCancel != nil {
		c.inviteCancel()
		c.inviteCancel = nil
	}
	if c.advertise != nil {
		c.advertise.Shutdown()
		c.advertise = nil
	}
	c.current = invitations.Invite{}
}

func inviteTXT(invite invitations.Invite) []string {
	txt := []string{
		"v=" + txtVersion,
		"channel=" + invite.Channel,
		"invite_id=" + invite.InviteID,
		"org_id=" + invite.OrganisationID,
		"org_name=" + invite.OrganisationName,
		"device_name=" + invite.DeviceName,
		"peer_id=" + invite.PeerID,
		"public_key=" + invite.PublicKey,
		"join_mode=" + invite.JoinMode,
		"target_user_id=" + invite.TargetUserID,
		"verification_hash=" + invite.VerificationHash,
		"expires_at=" + strconv.FormatInt(invite.ExpiresAt.Unix(), 10),
		"p2p_port=" + strconv.Itoa(invite.Port),
	}
	for i, addr := range invite.SwarmionAddrs {
		txt = append(txt, fmt.Sprintf("swarmion_addr_%d=%s", i, addr))
	}
	return txt
}

func parseTXTEntry(txt []string, hostName string, port int, ips []string) (invitations.NearbyInvite, bool) {
	values := map[string][]string{}
	for _, text := range txt {
		key, value, ok := strings.Cut(text, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		values[key] = append(values[key], value)
	}
	if first(values, "v") != txtVersion {
		return invitations.NearbyInvite{}, false
	}
	expiresAt := time.Unix(parseInt(first(values, "expires_at")), 0)
	item := invitations.NearbyInvite{
		InviteID:         first(values, "invite_id"),
		Channel:          invitations.ChannelMDNS,
		OrganisationID:   first(values, "org_id"),
		OrganisationName: first(values, "org_name"),
		DeviceName:       first(values, "device_name"),
		PeerID:           first(values, "peer_id"),
		PublicKey:        first(values, "public_key"),
		JoinMode:         invitations.NormalizeInviteJoinMode(first(values, "join_mode")),
		TargetUserID:     first(values, "target_user_id"),
		VerificationHash: first(values, "verification_hash"),
		P2PAddrs:         dedupeStrings(values["p2p_addr"]),
		SwarmionAddrs:    swarmionAddrsFromTXT(values),
		ExpiresAt:        expiresAt,
		HostName:         hostName,
		Port:             port,
		IPs:              dedupeStrings(ips),
	}
	if !invitations.ValidInviteJoinMode(item.JoinMode) {
		return invitations.NearbyInvite{}, false
	}
	if item.InviteID == "" ||
		item.OrganisationID == "" ||
		item.OrganisationName == "" ||
		item.PeerID == "" ||
		item.PublicKey == "" ||
		item.VerificationHash == "" ||
		len(item.SwarmionAddrs) == 0 {
		return invitations.NearbyInvite{}, false
	}
	return item, true
}

func swarmionAddrsFromTXT(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.HasPrefix(key, "swarmion_addr_") {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return txtIndex(keys[i], "swarmion_addr_") < txtIndex(keys[j], "swarmion_addr_")
	})
	var addrs []string
	for _, key := range keys {
		addrs = append(addrs, values[key]...)
	}
	return dedupeStrings(addrs)
}

func txtIndex(key string, prefix string) int {
	value := strings.TrimPrefix(key, prefix)
	index, err := strconv.Atoi(value)
	if err != nil {
		return 1 << 30
	}
	return index
}

func first(values map[string][]string, key string) string {
	for _, value := range values[key] {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseInt(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func cacheKey(item invitations.NearbyInvite) string {
	return item.Channel + "/" + item.OrganisationID + "/" + item.PeerID + "/" + invitations.NormalizeInviteJoinMode(item.JoinMode)
}

func inviteInstanceName(invite invitations.Invite) string {
	name := strings.TrimSpace(invite.OrganisationName)
	if name == "" {
		name = "organisation"
	}
	suffix := invite.InviteID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return "Protos " + name + " " + suffix
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
