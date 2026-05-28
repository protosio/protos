package main

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestRouteEnableArgsParsesTrailingDNSServerFlag(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := set.Parse([]string{"usa-exit-ash", "--dns-server", "1.1.1.1"}); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	instance, dnsServer, cidrs, err := routeEnableArgs(cli.NewContext(app, set, nil))
	if err != nil {
		t.Fatalf("routeEnableArgs: %v", err)
	}
	if instance != "usa-exit-ash" {
		t.Fatalf("instance = %q, want usa-exit-ash", instance)
	}
	if dnsServer != "1.1.1.1" {
		t.Fatalf("dnsServer = %q, want 1.1.1.1", dnsServer)
	}
	if len(cidrs) != 0 {
		t.Fatalf("cidrs = %v, want none", cidrs)
	}
}

func TestRouteEnableArgsUsesParsedFlag(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("dns-server", "", "")
	if err := set.Parse([]string{"--dns-server", "1.1.1.1", "usa-exit-ash"}); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	instance, dnsServer, cidrs, err := routeEnableArgs(cli.NewContext(app, set, nil))
	if err != nil {
		t.Fatalf("routeEnableArgs: %v", err)
	}
	if instance != "usa-exit-ash" {
		t.Fatalf("instance = %q, want usa-exit-ash", instance)
	}
	if dnsServer != "1.1.1.1" {
		t.Fatalf("dnsServer = %q, want 1.1.1.1", dnsServer)
	}
	if len(cidrs) != 0 {
		t.Fatalf("cidrs = %v, want none", cidrs)
	}
}

func TestRouteEnableArgsParsesTrailingCIDRFlags(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := set.Parse([]string{"usa-exit-ash", "--cidr", "1.2.3.4/32", "--route=203.0.113.0/24"}); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	instance, dnsServer, cidrs, err := routeEnableArgs(cli.NewContext(app, set, nil))
	if err != nil {
		t.Fatalf("routeEnableArgs: %v", err)
	}
	if instance != "usa-exit-ash" {
		t.Fatalf("instance = %q, want usa-exit-ash", instance)
	}
	if dnsServer != "" {
		t.Fatalf("dnsServer = %q, want empty", dnsServer)
	}
	want := []string{"1.2.3.4/32", "203.0.113.0/24"}
	if len(cidrs) != len(want) {
		t.Fatalf("cidrs = %v, want %v", cidrs, want)
	}
	for i := range want {
		if cidrs[i] != want[i] {
			t.Fatalf("cidrs = %v, want %v", cidrs, want)
		}
	}
}
