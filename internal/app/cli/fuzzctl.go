// Copyright (c) 2020, Sylabs, Inc. All rights reserved.

package cli

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/sylabs/compute-cli/internal/pkg/compute"
	"github.com/sylabs/compute-cli/internal/pkg/config"
	"golang.org/x/oauth2"
)

var (
	c *compute.Client

	tokenSrc oauth2.TokenSource
	cfg      *config.Config
)

// fuzzctl flag variables
var (
	debug bool

	httpAddr string
)

var FuzzctlCmd = &cobra.Command{
	Use:   "fuzzctl",
	Short: "fuzzctl enables control of workflows for Fuzzball.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// change log level if debugging
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		// Read configuration.
		cp, err := config.GetPath()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(path.Dir(cp), 0700); err != nil {
			return err
		}
		f, err := os.Open(cp)
		if err == nil {
			// Pre-existing config.
			defer f.Close()
			c, err := config.Read(f)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			cfg = c
		} else {
			// Write default config.
			c, err := config.Default()
			if err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			cfg = c
		}

		ctx := context.TODO()

		// Configure OAuth2 Token Source
		r, err := cfg.GetActiveRemote()
		if err != nil {
			return fmt.Errorf("failed to get active remote: %w", err)
		}

		switch t := r.GetAuthType(); t {
		case config.AuthConfigTypeAuthCodePKCE:
			tokenSrc = r.GetAuthCodePKCEConfig().TokenSource(ctx, r.GetToken())
		case config.AuthConfigTypeClientCredentials:
			tokenSrc = r.GetClientCredentialsConfig().TokenSource(ctx)
		default:
			return fmt.Errorf("unknown auth configuration type: %v", t)
		}

		// initialize global client for subcommands to leverage
		c = compute.NewClient(ctx, tokenSrc, httpAddr)

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		ar, err := cfg.GetActiveRemote()
		if err != nil {
			return err
		}

		if tokenSrc != nil {
			// Get updated token.
			t, err := tokenSrc.Token()
			if err != nil {
				return err
			}
			ar.SetToken(t)
		} else {
			ar.SetToken(nil)
		}

		// Save config.
		cp, err := config.GetPath()
		if err != nil {
			return err
		}
		f, err := os.OpenFile(cp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		return cfg.Write(f)
	},
}

func init() {
	FuzzctlCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	FuzzctlCmd.PersistentFlags().StringVar(&httpAddr, "http_addr", "http://localhost:8080", "Address to reach compute server")

	FuzzctlCmd.AddCommand(createCmd)
	FuzzctlCmd.AddCommand(deleteCmd)
	FuzzctlCmd.AddCommand(infoCmd)
	FuzzctlCmd.AddCommand(listCmd)
	FuzzctlCmd.AddCommand(serverInfoCmd)
	FuzzctlCmd.AddCommand(loginCmd)
	FuzzctlCmd.AddCommand(logoutCmd)
}