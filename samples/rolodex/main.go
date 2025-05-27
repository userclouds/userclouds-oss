package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type rolodexOption func() (done bool, err error)
type rolodexOptions map[string]rolodexOption

func (ro *rolodexOptions) addOption(key string, description string, option rolodexOption) {
	fmt.Printf("%s\t%s\n", key, description)
	(*ro)[key] = option
}

func getCreateContactInfo() contact {
	c := newContact()

	for {
		if name := getInput("Enter name"); name != "" {
			c.name = name
			break
		}
		showError("name cannot be empty")
	}
	for {
		if phoneNumber := getInput("Enter phone number"); phoneNumber != "" {
			c.phoneNumber = phoneNumber
			break
		}
		showError("phone number cannot be empty")
	}
	for {
		addMarketing := strings.ToUpper(getInput("Use for marketing? [Y/N]"))
		if addMarketing == "Y" {
			c.addPhoneNumberPurpose("rolodex_marketing")
			break
		}
		if addMarketing == "N" {
			break
		}
		showError("enter Y or N")
	}
	for {
		addSecurity := strings.ToUpper(getInput("Use for security? [Y/N]"))
		if addSecurity == "Y" {
			c.addPhoneNumberPurpose("rolodex_security")
			break
		}
		if addSecurity == "N" {
			break
		}
		showError("enter Y or N")
	}
	return c
}

func getInput(prompt string) string {
	fmt.Printf("\n----- %s > ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

func getMarketingMessage() string {
	for {
		if message := getInput("Enter marketing message"); message != "" {
			return message
		}
		showError("marketing message cannot be empty")
	}
}

func getOption(options rolodexOptions) rolodexOption {
	for {
		input := getInput("Pick an option")
		if option, found := options[strings.ToUpper(input)]; found {
			return option
		}
		showError(fmt.Sprintf("option unrecognized: %s", input))
	}
}

func showError(message string) {
	fmt.Printf("\n***** %s *****\n", message)
}

func showOptions(r *rolodex) rolodexOptions {
	fmt.Printf("\n*** Rolodex ***\n\n")

	options := rolodexOptions{}

	options.addOption(
		"C",
		"Create contact",
		func() (bool, error) {
			return false, r.createContact(getCreateContactInfo())
		},
	)

	options.addOption(
		"L",
		"List contacts",
		func() (bool, error) {
			return false, r.listContacts()
		},
	)

	options.addOption(
		"M",
		"Send marketing message w/ marketing client purpose",
		func() (bool, error) {
			return false, r.sendMarketingMessage(getMarketingMessage(), "api_key_for_marketing")
		},
	)

	options.addOption(
		"S",
		"Send marketing message w/ security client purpose",
		func() (bool, error) {
			return false, r.sendMarketingMessage(getMarketingMessage(), "api_key_for_operations")
		},
	)

	options.addOption(
		"Q",
		"Quit",
		func() (bool, error) {
			return true, nil
		},
	)

	return options
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	rdex, err := newRolodex(context.Background())
	if err != nil {
		panic(err)
	}
	defer rdex.teardown()

	for {
		option := getOption(showOptions(rdex))
		done, err := option()
		if err != nil {
			panic(err)
		}
		if done {
			break
		}
	}
}
