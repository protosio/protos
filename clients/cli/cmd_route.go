package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var cmdRoute *cli.Command = &cli.Command{
	Name:    "route",
	Aliases: []string{"exit"},
	Usage:   "Route local traffic through a Protos instance",
	Subcommands: []*cli.Command{
		{
			Name:      "status",
			ArgsUsage: "[instance]",
			Usage:     "Show configured exit routes",
			Action: func(c *cli.Context) error {
				return routeStatus(c.Args().Get(0))
			},
		},
		{
			Name:      "enable",
			Aliases:   []string{"set"},
			ArgsUsage: "<instance>",
			Usage:     "Route local CIDRs through an instance public IP",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:    "cidr",
					Aliases: []string{"route"},
					Usage:   "Route this network CIDR through the exit instance; repeat for multiple CIDRs; defaults to full tunnel",
				},
				&cli.StringFlag{
					Name:  "dns-server",
					Usage: "Forward non-Protos DNS queries to this public resolver while the exit route is active",
				},
			},
			Action: func(c *cli.Context) error {
				instance, dnsServer, cidrs, err := routeEnableArgs(c)
				if err != nil {
					return err
				}
				if instance == "" {
					return showSubcommandHelp(c)
				}
				return enableExitRoute(instance, dnsServer, cidrs)
			},
		},
		{
			Name:    "disable",
			Aliases: []string{"clear", "off"},
			Usage:   "Stop routing local traffic through an instance",
			Action: func(c *cli.Context) error {
				return disableExitRoute()
			},
		},
	},
}

func routeEnableArgs(c *cli.Context) (string, string, []string, error) {
	if c == nil {
		return "", "", nil, fmt.Errorf("cli context is required")
	}
	dnsServer := c.String("dns-server")
	cidrs := append([]string(nil), c.StringSlice("cidr")...)
	instance := ""
	args := c.Args().Slice()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--cidr" || arg == "--route":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("%s requires a value", arg)
			}
			cidrs = append(cidrs, args[i+1])
			i++
		case strings.HasPrefix(arg, "--cidr="):
			cidrs = append(cidrs, strings.TrimPrefix(arg, "--cidr="))
		case strings.HasPrefix(arg, "--route="):
			cidrs = append(cidrs, strings.TrimPrefix(arg, "--route="))
		case arg == "--dns-server":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("--dns-server requires a value")
			}
			dnsServer = args[i+1]
			i++
		case strings.HasPrefix(arg, "--dns-server="):
			dnsServer = strings.TrimPrefix(arg, "--dns-server=")
		case strings.HasPrefix(arg, "-"):
			return "", "", nil, fmt.Errorf("unknown route enable flag %s", arg)
		case instance == "":
			instance = arg
		default:
			return "", "", nil, fmt.Errorf("unexpected route enable argument %s", arg)
		}
	}
	return instance, dnsServer, cidrs, nil
}

func routeStatus(instance string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetExitRoutes(ctx, &pbApic.GetExitRoutesRequest{Instance: instance})
	if err != nil {
		return fmt.Errorf("failed to retrieve exit routes: %w", err)
	}

	if instance != "" {
		fmt.Printf("Target: %s\n", instance)
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t", "Device", "Instance", "Name", "Public IP", "Location", "CIDRs", "DNS", "Status")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t", "------", "--------", "----", "---------", "--------", "-----", "---", "------")
	for _, route := range resp.Routes {
		dnsServer := route.DnsServer
		if dnsServer == "" {
			dnsServer = "default"
		}
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t", route.DeviceId, route.InstanceId, route.InstanceName, route.PublicIp, route.Location, strings.Join(route.Cidrs, ","), dnsServer, route.Status)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func enableExitRoute(instance string, dnsServer string, cidrs []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.SetExitRoute(ctx, &pbApic.SetExitRouteRequest{Instance: instance, DnsServer: dnsServer, Cidrs: cidrs})
	if err != nil {
		return fmt.Errorf("failed to enable exit route through '%s': %w", instance, err)
	}
	route := resp.Route
	fmt.Printf("Routing local traffic through %s (%s)\n", route.InstanceName, route.PublicIp)
	if len(route.Cidrs) > 0 {
		fmt.Printf("Routed CIDRs: %s\n", strings.Join(route.Cidrs, ", "))
	}
	if route.DnsServer != "" {
		fmt.Printf("Forwarding external DNS through %s\n", route.DnsServer)
	}
	return nil
}

func disableExitRoute() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.ClearExitRoute(ctx, &pbApic.ClearExitRouteRequest{}); err != nil {
		return fmt.Errorf("failed to disable exit route: %w", err)
	}
	fmt.Println("Exit route disabled")
	return nil
}
