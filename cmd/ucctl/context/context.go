package context

import (
	stdcontext "context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/secret/prefix"
)

// getContextNames returns a sorted list of available context names for completion
func getContextNames() []string {
	cfg, err := common.Load("")
	if err != nil {
		return []string{}
	}

	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidContextArgs provides dynamic completion for context names
func ValidContextArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete the first argument (context name)
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContextNames(), cobra.ShellCompDirectiveNoFileComp
}

// ListCommand lists all contexts
type ListCommand struct{}

func (c *ListCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := common.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts configured")
		return nil
	}

	// Get sorted context names
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print contexts in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "CURRENT\tNAME\tURL")

	for _, name := range names {
		ctx := cfg.Contexts[name]
		current := " "
		if name == cfg.CurrentContext {
			current = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", current, name, ctx.URL)
	}

	return w.Flush()
}

// UseCommand switches to a different context
type UseCommand struct{}

func (c *UseCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context use <context-name>")
	}

	contextName := args[0]

	cfg, err := common.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.UseContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Switched to context %q\n", contextName)
	return nil
}

// SetCommand creates or updates a context
type SetCommand struct {
	URL      string
	ClientID string
	// TODO: this should be a secret string.
	ClientSecret string
}

func (c *SetCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context set <context-name> --url <url> --client-id <id> --client-secret <secret>")
	}

	contextName := args[0]

	// Validate required fields
	if c.URL == "" {
		return fmt.Errorf("--url is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("--client-id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("--client-secret is required")
	}

	cfg, err := common.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse and validate the client secret
	var clientSecret secret.String
	if hasSecretPrefix(c.ClientSecret) {
		// If it has a prefix, use it as a location and validate
		clientSecret = *secret.FromLocation(c.ClientSecret)
		if err := clientSecret.Validate(); err != nil {
			return fmt.Errorf("invalid client secret: %w", err)
		}
	} else {
		// If no prefix, wrap it as a dev-literal secret
		clientSecret = secret.NewTestString(c.ClientSecret)
	}

	ctx := &common.Context{
		URL:          c.URL,
		ClientID:     c.ClientID,
		ClientSecret: clientSecret,
	}

	cfg.SetContext(contextName, ctx)

	// If this is the first context, make it current
	if len(cfg.Contexts) == 1 {
		cfg.CurrentContext = contextName
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context %q set\n", contextName)
	if cfg.CurrentContext == contextName {
		fmt.Printf("Switched to context %q\n", contextName)
	}

	return nil
}

// DeleteCommand deletes a context
type DeleteCommand struct{}

func (c *DeleteCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context delete <context-name>")
	}

	contextName := args[0]

	cfg, err := common.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.DeleteContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context %q deleted\n", contextName)
	return nil
}

// ShowCommand displays the current context
type ShowCommand struct{}

func (c *ShowCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := common.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine which context to show
	var contextName string
	var ctx *common.Context

	if len(args) > 0 {
		// Show the specified context
		contextName = args[0]
		ctx, err = cfg.GetContext(contextName)
		if err != nil {
			return err
		}
	} else {
		// Show the current context
		if cfg.CurrentContext == "" {
			fmt.Println("No current context set")
			return nil
		}
		contextName = cfg.CurrentContext
		ctx, err = cfg.GetCurrentContext()
		if err != nil {
			return err
		}
	}

	// Display context information
	if contextName == cfg.CurrentContext {
		fmt.Printf("Current context: %s\n", contextName)
	} else {
		fmt.Printf("Context: %s\n", contextName)
	}

	fmt.Printf("URL: %s\n", ctx.URL)
	fmt.Printf("Client ID: %s\n", ctx.ClientID)

	// Display the masked client secret value
	secretLocationBytes, _ := ctx.ClientSecret.MarshalText()
	secretLocation := string(secretLocationBytes)

	// Try to resolve and mask the secret
	maskedSecret, err := maskSecretWithSuffix(&ctx.ClientSecret)
	if err != nil {
		// If we can't resolve, just show the masked location
		fmt.Printf("Client Secret: ****\n")
	} else {
		fmt.Printf("Client Secret: %s\n", maskedSecret)
	}

	// If it's an external secret, also show the location
	if secretLocation != "" && !isPlainSecret(secretLocation) {
		fmt.Printf("Secret Location: %s\n", secretLocation)
	}

	return nil
}

// isPlainSecret checks if the secret location is a plain secret (no prefix or dev-literal)
func isPlainSecret(location string) bool {
	return location == "" ||
		!hasSecretPrefix(location) ||
		startsWithPrefix(location, "dev-literal://")
}

func hasSecretPrefix(location string) bool {
	_, err := prefix.PrefixFromString(location)
	return err == nil
}

// startsWithPrefix checks if s starts with prefix
func startsWithPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// maskSecretWithSuffix masks a secret but shows the last 4 characters for reference
func maskSecretWithSuffix(s *secret.String) (string, error) {
	// Get the secret location to check if it's external
	secretLocationBytes, _ := s.MarshalText()
	secretLocation := string(secretLocationBytes)

	// For external secrets (env://, aws://, kube://), resolve them first
	if secretLocation != "" && !isPlainSecret(secretLocation) && !startsWithPrefix(secretLocation, "dev-literal://") {
		// Resolve the actual secret value
		ctx := stdcontext.Background()
		resolved, err := s.Resolve(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to resolve secret: %w", err)
		}
		return maskSecret(resolved), nil
	}

	// For dev-literal://, strip the prefix and show last 4 chars
	secretValue := secretLocation
	if startsWithPrefix(secretValue, "dev-literal://") {
		secretValue = secretValue[len("dev-literal://"):]
	}

	if len(secretValue) <= 4 {
		return "****", nil
	}
	return "****" + secretValue[len(secretValue)-4:], nil
}

// maskSecret masks all but the last 4 characters of a secret
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}
