package main

import (
	"github.com/hashicorp/terraform/helper/schema"
	hyperkit "github.com/moby/hyperkit/go"
)

// Config contains configuration for the Terraform provider.
type Config struct {
	HyperKitBin  string
	VPNKitSocket string
	StateDir     string
	Console      int
}

// Provider returns a Terraform resource provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"hyperkit_bin": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The path to the hyperkit binary.",
			},
			"vpnkit_socket": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "auto",
				Description: "The path to the vpnkit socket. If set to auto hyperkit will attempt to use the existing Docker vpnkit network.",
			},
			"state_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The path to the directory hyperkit will store vm state in. If empty hyperkit will not write to disk.",
			},
			"console": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "stdio",
				Description: "[stdio, file, log]",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"hyperkit_vm": resourceHyperKitVM(),
		},
		ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {
			consoles := map[string]int{
				"stdio": hyperkit.ConsoleStdio,
				"file":  hyperkit.ConsoleFile,
				"log":   hyperkit.ConsoleLog,
			}

			return Config{
				HyperKitBin:  d.Get("hyperkit_bin").(string),
				VPNKitSocket: d.Get("vpnkit_socket").(string),
				StateDir:     d.Get("state_dir").(string),
				Console:      consoles[d.Get("console").(string)],
			}, nil
		},
	}
}
