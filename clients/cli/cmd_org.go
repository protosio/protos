package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

const (
	defaultInviteChannel    = "mdns"
	inviteJoinModeAny       = "any"
	inviteJoinModeNewUser   = "new_user"
	inviteJoinModeNewDevice = "new_device"
)

var cmdOrg *cli.Command = &cli.Command{
	Name:    "org",
	Aliases: []string{"organization"},
	Usage:   "Manage organization membership",
	Subcommands: []*cli.Command{
		{
			Name:  "invite",
			Usage: "Start an invite for a new user or another device",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "organization-id",
					Aliases: []string{"org-id", "organisation-id"},
					Usage:   "Organization ID to invite into; defaults to the local organization",
				},
				&cli.StringFlag{
					Name:  "channel",
					Value: defaultInviteChannel,
					Usage: "Invite channel to advertise on",
				},
				&cli.StringFlag{
					Name:    "mode",
					Aliases: []string{"join-mode"},
					Value:   inviteJoinModeNewDevice,
					Usage:   "Invite mode: new-user or new-device",
				},
				&cli.StringFlag{
					Name:  "username",
					Usage: "Existing username to target for new-device invites; defaults to the local user",
				},
			},
			Action: func(c *cli.Context) error {
				return startOrgInvite(c)
			},
		},
		{
			Name:    "nearby",
			Aliases: []string{"ls"},
			Usage:   "List nearby organization invites",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "channel",
					Value: defaultInviteChannel,
					Usage: "Invite channel to scan",
				},
			},
			Action: func(c *cli.Context) error {
				return listNearbyOrganisations(c)
			},
		},
		{
			Name:  "join",
			Usage: "Join an organization as a new user or as another device for an existing user",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "username",
					Usage: "Username to create for new-user joins; optional for new-device joins with targeted invites",
				},
				&cli.StringFlag{
					Name:  "name",
					Usage: "Display name for a newly-created user; defaults to username",
				},
				&cli.StringFlag{
					Name:     "verification-code",
					Aliases:  []string{"code"},
					Required: true,
					Usage:    "Verification code shown by the inviting device",
				},
				&cli.StringFlag{
					Name:    "organization-id",
					Aliases: []string{"org-id", "organisation-id"},
					Usage:   "Organization ID from org nearby; scanned automatically if omitted",
				},
				&cli.StringFlag{
					Name:  "peer-id",
					Usage: "Inviting peer ID from org nearby; scanned automatically if omitted",
				},
				&cli.StringFlag{
					Name:  "invite-id",
					Usage: "Invite ID from org nearby; optional when there is one nearby invite",
				},
				&cli.StringFlag{
					Name:  "channel",
					Value: defaultInviteChannel,
					Usage: "Invite channel to scan or join through",
				},
				&cli.StringFlag{
					Name:    "mode",
					Aliases: []string{"join-mode"},
					Usage:   "Join mode: new-user or new-device; defaults to the selected invite",
				},
				&cli.BoolFlag{
					Name:  "no-input",
					Usage: "Fail instead of prompting when multiple nearby invites match",
				},
			},
			Action: func(c *cli.Context) error {
				return joinOrganisation(c)
			},
		},
	},
}

func startOrgInvite(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	joinMode, err := inviteJoinMode(c.String("mode"))
	if err != nil {
		return err
	}
	resp, err := client.StartDeviceInvite(ctx, &pbApic.StartDeviceInviteRequest{
		OrganisationId: strings.TrimSpace(c.String("organization-id")),
		Channel:        inviteChannel(c),
		JoinMode:       joinMode,
		Username:       strings.TrimSpace(c.String("username")),
	})
	if err != nil {
		return fmt.Errorf("failed to start organization invite: %w", err)
	}

	fmt.Printf("Invite ID: %s\n", resp.InviteId)
	fmt.Printf("Channel: %s\n", resp.Channel)
	fmt.Printf("Mode: %s\n", joinModeLabel(resp.JoinMode))
	fmt.Printf("Verification code: %s\n", resp.VerificationCode)
	if resp.ExpiresAtUnix > 0 {
		fmt.Printf("Expires: %s\n", time.Unix(resp.ExpiresAtUnix, 0).Local().Format(time.RFC3339))
	}
	if resp.AdvertiseName != "" {
		fmt.Printf("Advertised as: %s\n", resp.AdvertiseName)
	}
	if resp.AdvertiseService != "" {
		fmt.Printf("Service: %s\n", resp.AdvertiseService)
	}

	return nil
}

func listNearbyOrganisations(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListNearbyOrganisations(ctx, &pbApic.ListNearbyOrganisationsRequest{
		Channel: inviteChannel(c),
	})
	if err != nil {
		return fmt.Errorf("failed to list nearby organization invites: %w", err)
	}

	printNearbyOrganisations(resp.Organisations)
	return nil
}

func joinOrganisation(c *cli.Context) error {
	req := joinOrganisationRequestFromFlags(c)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if req.OrganisationId == "" || req.PeerId == "" {
		if err := fillJoinRequestFromNearby(ctx, req, c.Bool("no-input")); err != nil {
			return err
		}
	}
	if req.JoinMode == inviteJoinModeNewUser && req.Username == "" {
		return fmt.Errorf("username is required for new-user joins")
	}

	_, err := client.JoinOrganisation(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to join organization: %w", err)
	}
	if req.Username != "" {
		fmt.Printf("Joined organization %s as %s\n", req.OrganisationId, req.Username)
	} else {
		fmt.Printf("Joined organization %s\n", req.OrganisationId)
	}
	return nil
}

func joinOrganisationRequestFromFlags(c *cli.Context) *pbApic.JoinOrganisationRequest {
	username := strings.TrimSpace(c.String("username"))
	name := strings.TrimSpace(c.String("name"))
	if name == "" {
		name = username
	}
	return &pbApic.JoinOrganisationRequest{
		OrganisationId:   strings.TrimSpace(c.String("organization-id")),
		PeerId:           strings.TrimSpace(c.String("peer-id")),
		InviteId:         strings.TrimSpace(c.String("invite-id")),
		Username:         username,
		Name:             name,
		Channel:          inviteChannel(c),
		VerificationCode: strings.TrimSpace(c.String("verification-code")),
		JoinMode:         normalizeJoinMode(c.String("mode")),
	}
}

func fillJoinRequestFromNearby(ctx context.Context, req *pbApic.JoinOrganisationRequest, noInput bool) error {
	resp, err := client.ListNearbyOrganisations(ctx, &pbApic.ListNearbyOrganisationsRequest{
		Channel: req.Channel,
	})
	if err != nil {
		return fmt.Errorf("failed to scan nearby organization invites: %w", err)
	}

	candidates := matchingNearbyOrganisations(resp.Organisations, req)
	if len(candidates) == 0 {
		return fmt.Errorf("no nearby organization invite matched the supplied filters")
	}

	selected := candidates[0]
	if len(candidates) > 1 {
		if noInput {
			return fmt.Errorf("multiple nearby organization invites matched; pass --organization-id and --peer-id")
		}
		selected, err = promptNearbyOrganisation(candidates)
		if err != nil {
			return err
		}
	}

	req.OrganisationId = selected.OrganisationId
	req.PeerId = selected.PeerId
	req.InviteId = selected.InviteId
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = selected.Channel
	}
	if strings.TrimSpace(req.JoinMode) == "" && strings.TrimSpace(selected.JoinMode) != "" {
		req.JoinMode = normalizeJoinMode(selected.JoinMode)
	}
	return nil
}

func matchingNearbyOrganisations(items []*pbApic.NearbyOrganisation, req *pbApic.JoinOrganisationRequest) []*pbApic.NearbyOrganisation {
	matches := make([]*pbApic.NearbyOrganisation, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if req.OrganisationId != "" && item.OrganisationId != req.OrganisationId {
			continue
		}
		if req.PeerId != "" && item.PeerId != req.PeerId {
			continue
		}
		if req.InviteId != "" && item.InviteId != req.InviteId {
			continue
		}
		if req.Channel != "" && item.Channel != "" && item.Channel != req.Channel {
			continue
		}
		if req.JoinMode != "" && !nearbyInviteSupportsJoinMode(item, req.JoinMode) {
			continue
		}
		matches = append(matches, item)
	}
	return matches
}

func promptNearbyOrganisation(items []*pbApic.NearbyOrganisation) (*pbApic.NearbyOrganisation, error) {
	options := make([]string, 0, len(items))
	byOption := map[string]*pbApic.NearbyOrganisation{}
	for i, item := range items {
		option := fmt.Sprintf("%d. %s", i+1, nearbyOrganisationLabel(item))
		options = append(options, option)
		byOption[option] = item
	}

	selected := ""
	if err := survey.AskOne(surveySelect(options, "Select organization invite"), &selected); err != nil {
		return nil, err
	}
	item, ok := byOption[selected]
	if !ok {
		return nil, fmt.Errorf("selected organization invite was not found")
	}
	return item, nil
}

func printNearbyOrganisations(items []*pbApic.NearbyOrganisation) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t", "Organization", "Device", "Mode", "Peer ID", "Invite ID", "Channel")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", "------------", "------", "----", "-------", "---------", "-------")
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", item.OrganisationName, item.DeviceName, joinModeLabel(item.JoinMode), item.PeerId, item.InviteId, item.Channel)
	}
	fmt.Fprint(w, "\n")
}

func nearbyOrganisationLabel(item *pbApic.NearbyOrganisation) string {
	if item == nil {
		return ""
	}
	parts := []string{}
	if item.OrganisationName != "" {
		parts = append(parts, item.OrganisationName)
	} else {
		parts = append(parts, item.OrganisationId)
	}
	if item.DeviceName != "" {
		parts = append(parts, "from "+item.DeviceName)
	}
	if item.PeerId != "" {
		parts = append(parts, "peer "+item.PeerId)
	}
	if item.InviteId != "" {
		parts = append(parts, "invite "+item.InviteId)
	}
	if item.JoinMode != "" {
		parts = append(parts, "mode "+joinModeLabel(item.JoinMode))
	}
	if item.Channel != "" {
		parts = append(parts, "via "+item.Channel)
	}
	return strings.Join(parts, " | ")
}

func inviteChannel(c *cli.Context) string {
	channel := strings.TrimSpace(c.String("channel"))
	if channel == "" {
		return defaultInviteChannel
	}
	return channel
}

func inviteJoinMode(value string) (string, error) {
	joinMode := normalizeJoinMode(value)
	if joinMode == "" {
		joinMode = inviteJoinModeNewDevice
	}
	switch joinMode {
	case inviteJoinModeNewUser, inviteJoinModeNewDevice:
		return joinMode, nil
	default:
		return "", fmt.Errorf("invite mode must be new-user or new-device")
	}
}

func normalizeJoinMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	switch value {
	case "", inviteJoinModeAny:
		return value
	case "user", "newuser", "new_user":
		return inviteJoinModeNewUser
	case "device", "newdevice", "new_device":
		return inviteJoinModeNewDevice
	default:
		return value
	}
}

func nearbyInviteSupportsJoinMode(item *pbApic.NearbyOrganisation, requestedMode string) bool {
	requestedMode = normalizeJoinMode(requestedMode)
	if requestedMode == "" || requestedMode == inviteJoinModeAny {
		return true
	}
	itemMode := normalizeJoinMode(item.GetJoinMode())
	return itemMode == "" || itemMode == inviteJoinModeAny || itemMode == requestedMode
}

func joinModeLabel(value string) string {
	switch normalizeJoinMode(value) {
	case inviteJoinModeNewUser:
		return "new user"
	case inviteJoinModeNewDevice:
		return "new device"
	case inviteJoinModeAny, "":
		return "any"
	default:
		return value
	}
}
