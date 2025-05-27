package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
)

// Text strings for the demo, moved here for readability of the code
const textBlock1 = `
Welcome to the Tokenizer Sample App! This app demonstrates the basics of Tokenizer
by tokenizing a secret message. Who would you like to send a secret message to?
`
const textBlock2 = `
Great, let's create an access policy that gives only %s access to certain messages
We'll call it %s.
`
const textBlock3 = `
Access policy created:
%+v

Now let's create a message: what do you want your secret message to say?
`
const textBlock4 = `
Cool. Now we'll encode our secret message by creating a token for it. We'll use
%s so that only %s can read the Message
`
const textBlock5 = `
Your token (referencing your message): %s

Great. Let's test resolving the token and reading the message.
`
const textBlock6 = `
What user would you like to try resolving the message as? (Try "%s" or someone else)
`

func demoTokenizer() (quit bool) {
	ctx := context.Background()

	// "Welcome to the Tokenizer Sample App!..."
	fmt.Print(textBlock1)

	tenantURL := os.Getenv("USERCLOUDS_TENANT_URL")
	clientID := os.Getenv("USERCLOUDS_CLIENT_ID")
	clientSecret := os.Getenv("USERCLOUDS_CLIENT_SECRET")

	if tenantURL == "" || clientID == "" || clientSecret == "" {
		log.Fatal("missing one or more required environment variables: USERCLOUDS_TENANT_URL, USERCLOUDS_CLIENT_ID, USERCLOUDS_CLIENT_SECRET")
	}

	ts, err := jsonclient.ClientCredentialsForURL(tenantURL, clientID, clientSecret, nil)
	if err != nil {
		log.Fatalf("Error getting token. check values of provided client id, client secret and tenant url : %s", err)
	}

	recipient := getInput("Recipient (default: Bob)", "Bob")
	policyName := fmt.Sprintf("AccessPolicyUser%s-%s", recipient, uuid.Must(uuid.NewV4()))

	// "Great, let's create an access policy..."
	fmt.Printf(textBlock2, recipient, policyName)
	if ok := getInput("OK? [Yn]", "Y"); ok == "n" {
		log.Fatal("Aborting")
	}

	client := idp.NewTokenizerClient(tenantURL, idp.JSONClient(ts))
	templateDef := policy.AccessPolicyTemplate{
		Name:     policyName + "Template",
		Function: `function policy(context, params) { if (context.client.user === params.recipient) { return true; } return false; }`,
	}
	apt, err := client.CreateAccessPolicyTemplate(ctx, templateDef)
	if err != nil {
		log.Fatalf("Error creating access policy template: %s", err)
	}

	// Clean up this template so this demo can re-create it again when run again (a production app wouldn't do this)
	defer deleteAccessPolicyTemplate(ctx, client, apt.ID)

	policyDef := policy.AccessPolicy{
		Name:       policyName,
		Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt.ID}, TemplateParameters: fmt.Sprintf(`{"recipient": "%s"}`, recipient)}},
		PolicyType: policy.PolicyTypeCompositeAnd,
	}
	ap, err := client.CreateAccessPolicy(ctx, policyDef)
	if err != nil {
		log.Fatalf("Error creating access policy: %s", err)
	}

	jsonAP, err := json.MarshalIndent(&ap, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling access policy to JSON: %s", err)
	}

	// "Access policy created...Now let's create a message..."
	fmt.Printf(textBlock3, jsonAP)
	message := getInput("Message (default: Hello World!)", "Hello World!")

	// "Cool. Now we'll encode our secret message..."
	fmt.Printf(textBlock4, policyName, recipient)
	token, err := client.CreateToken(ctx, message, userstore.ResourceID{ID: policy.TransformerUUID.ID}, userstore.ResourceID{ID: ap.ID})
	if err != nil {
		log.Fatalf("Error creating token: %s", err)
	}

	// Clean up this token and policy so this demo can re-create it again when run again (a production app wouldn't do this)
	defer func() {
		deleteToken(ctx, client, token)
		deleteAccessPolicy(ctx, client, ap.ID)
	}()

	// "Your token...Let's test resolving..."
	fmt.Printf(textBlock5, token)

	for {
		// "What user would you like to try resolving the message as?..."
		fmt.Printf(textBlock6, recipient)
		viewer := getInput("Viewing user", "")

		resolved, err := client.ResolveToken(ctx, token, policy.ClientContext{"user": viewer}, nil)
		if err != nil {
			log.Fatalf("Error resolving token: %s", err)
		}
		if resolved == "" {
			fmt.Println("\nToken not resolved.")
		} else {
			fmt.Printf("\nToken resolved: %s\n", resolved)
		}
		time.Sleep(time.Second)

		fmt.Printf("\n" +
			"What would you like to do next?\n" +
			"1 = Try resolving the message as someone else\n" +
			"2 = Start Again\n" +
			"3 = Exit\n")
		switch choice := getInput("Choice? (1, 2, or 3)", "1"); choice {
		case "1":
			continue
		case "2":
			return false
		case "3":
			return true
		default:
			continue
		}
	}
}

func getInput(prompt string, defaultValue string) string {
	fmt.Printf("%s: ", prompt)

	rdr := bufio.NewReader(os.Stdin)
	input, err := rdr.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading input: %s", err)
	}

	if s := strings.TrimSpace(input); s != "" {
		return s
	}
	return defaultValue
}

func deleteAccessPolicyTemplate(ctx context.Context, client *idp.TokenizerClient, id uuid.UUID) {
	if err := client.DeleteAccessPolicyTemplate(ctx, id, 0); err != nil {
		log.Fatalf("Error deleting access policy template: %s", err)
	}
}

func deleteAccessPolicy(ctx context.Context, client *idp.TokenizerClient, id uuid.UUID) {
	if err := client.DeleteAccessPolicy(ctx, id, 0); err != nil {
		log.Fatalf("Error deleting access policy: %s", err)
	}
}

func deleteToken(ctx context.Context, client *idp.TokenizerClient, token string) {
	if err := client.DeleteToken(ctx, token); err != nil {
		log.Fatalf("Error deleting token: %s", err)
	}
}

func main() {
	for {
		if quit := demoTokenizer(); quit {
			break
		}
		fmt.Println()
	}
}
