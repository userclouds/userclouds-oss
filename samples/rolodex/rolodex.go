package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
)

type rolodex struct {
	ctx             context.Context
	idpClient       *idp.Client
	tokenizerClient *idp.TokenizerClient

	phoneNumberColumn                   *userstore.Column
	marketingPurpose                    *userstore.Purpose
	securityPurpose                     *userstore.Purpose
	isAPIKeyAllowedAccessPolicyTemplate *policy.AccessPolicyTemplate
	isMarketingAPIKeyAccessPolicy       *policy.AccessPolicy
	isOperationsAPIKeyAccessPolicy      *policy.AccessPolicy
	getPhoneNumbersForMarketingAccessor *userstore.Accessor
	listContactsAccessor                *userstore.Accessor
	createPhoneNumberMutator            *userstore.Mutator
}

// We create a token source with the specified tenant URL and client ID and secret, and create IDP
// client and tokenizer clients. We then create the column, purposes, access policies, accessors,
// and mutators used for this sample.

func newRolodex(ctx context.Context) (*rolodex, error) {
	// create token source

	tenantURL := os.Getenv("UC_TENANT_BASE_URL")
	plexClientID := os.Getenv("PLEX_CLIENT_ID")
	plexClientSecret := os.Getenv("PLEX_CLIENT_SECRET")

	ts := jsonclient.ClientCredentialsTokenSource(tenantURL+"/oidc/token", plexClientID, plexClientSecret, nil)

	// create IDP client

	idpClient, err := idp.NewClient(tenantURL, idp.JSONClient(ts, jsonclient.StopLogging()))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// create tokenizer client

	tokenizerClient := idp.NewTokenizerClient(tenantURL, idp.JSONClient(ts, jsonclient.StopLogging()))

	r := rolodex{
		ctx:             ctx,
		idpClient:       idpClient,
		tokenizerClient: tokenizerClient,
	}

	// define columns

	phoneNumberColumn, err :=
		r.idpClient.CreateColumn(
			r.ctx,
			userstore.Column{
				Name:      "rolodex_phone_number",
				DataType:  datatype.String,
				IndexType: userstore.ColumnIndexTypeIndexed,
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.phoneNumberColumn = phoneNumberColumn

	// define purposes

	marketingPurpose, err :=
		r.idpClient.CreatePurpose(r.ctx,
			userstore.Purpose{
				Name:        "rolodex_marketing",
				Description: "This purpose is used to grant access to PII for marketing.",
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.marketingPurpose = marketingPurpose

	securityPurpose, err :=
		r.idpClient.CreatePurpose(r.ctx,
			userstore.Purpose{
				Name:        "rolodex_security",
				Description: "This purpose is used to grant access to PII for security reasons.",
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.securityPurpose = securityPurpose

	// define access policies

	// This access policy template expects the supplied client context to contain the expected api key for the template.

	isAPIKeyAllowedAccessPolicyTemplate, err :=
		r.tokenizerClient.CreateAccessPolicyTemplate(
			r.ctx,
			policy.AccessPolicyTemplate{
				Name:     "RolodexIsAPIKeyAllowedTemplate",
				Function: "function policy(context, params) { return context['client']['api_key'] === params.api_key }",
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.isAPIKeyAllowedAccessPolicyTemplate = isAPIKeyAllowedAccessPolicyTemplate

	// This access policy uses the previously specified access policy template parameterized with
	// an "api_key_for_marketing" api key.

	isMarketingAPIKeyAccessPolicy, err :=
		r.tokenizerClient.CreateAccessPolicy(
			r.ctx,
			policy.AccessPolicy{
				Name: "RolodexIsMarketingAPIKey",
				Components: []policy.AccessPolicyComponent{
					{
						Template: &userstore.ResourceID{
							ID: isAPIKeyAllowedAccessPolicyTemplate.ID,
						},
						TemplateParameters: `{"api_key": "api_key_for_marketing"}`,
					},
				},
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.isMarketingAPIKeyAccessPolicy = isMarketingAPIKeyAccessPolicy

	// This access policy uses the previously specified access policy template parameterized with
	// an "api_key_for_operations" api key.

	isOperationsAPIKeyAccessPolicy, err :=
		r.tokenizerClient.CreateAccessPolicy(
			r.ctx,
			policy.AccessPolicy{
				Name: "RolodexIsOperationsAPIKey",
				Components: []policy.AccessPolicyComponent{
					{
						Template: &userstore.ResourceID{
							ID: isAPIKeyAllowedAccessPolicyTemplate.ID,
						},
						TemplateParameters: `{"api_key": "api_key_for_operations"}`,
					},
				},
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.isOperationsAPIKeyAccessPolicy = isOperationsAPIKeyAccessPolicy

	// define accessors

	// This accessor expects the client context api key to be "api_key_for_operations", and returns the
	// "id", "name", and "rolodex_phone_number" columns if they have a an "operational" consented
	// purpose associated with them.

	listContactsAccessor, err :=
		r.idpClient.CreateAccessor(
			r.ctx,
			userstore.Accessor{
				ID:                 uuid.Nil,
				Name:               "RolodexListContacts",
				DataLifeCycleState: userstore.DataLifeCycleStateLive,
				Columns: []userstore.ColumnOutputConfig{
					{
						Column:      userstore.ResourceID{Name: "id"},
						Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					},
					{
						Column:      userstore.ResourceID{Name: "name"},
						Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					},
					{
						Column:      userstore.ResourceID{Name: "rolodex_phone_number"},
						Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					},
				},
				AccessPolicy: userstore.ResourceID{ID: isOperationsAPIKeyAccessPolicy.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: `{rolodex_phone_number} != ?`,
				},
				Purposes: []userstore.ResourceID{
					{Name: "operational"},
				},
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.listContactsAccessor = listContactsAccessor

	// This accessor expects the client context api key to be "api_key_for_marketing", and returns the
	// "rolodex_phone_number" columns if they have a "rolodex_marketing" consented purpose associated
	// with them. The selector ensures that only records with a non-empty "rolodex_phone_number" are
	// considered.

	getPhoneNumbersForMarketingAccessor, err :=
		r.idpClient.CreateAccessor(
			r.ctx,
			userstore.Accessor{
				ID:                 uuid.Nil,
				Name:               "RolodexGetPhoneNumbersForMarketing",
				DataLifeCycleState: userstore.DataLifeCycleStateLive,
				Columns: []userstore.ColumnOutputConfig{
					{
						Column:      userstore.ResourceID{Name: "rolodex_phone_number"},
						Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					},
				},
				AccessPolicy: userstore.ResourceID{ID: isMarketingAPIKeyAccessPolicy.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: `{rolodex_phone_number} != ?`,
				},
				Purposes: []userstore.ResourceID{
					{Name: "rolodex_marketing"},
				},
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.getPhoneNumbersForMarketingAccessor = getPhoneNumbersForMarketingAccessor

	// define mutators

	// This mutator creates a "rolodex_phone_number" column for the specified record.

	createPhoneNumberMutator, err :=
		r.idpClient.CreateMutator(
			r.ctx,
			userstore.Mutator{
				ID:   uuid.Nil,
				Name: "RolodexCreatePhoneNumber",
				Columns: []userstore.ColumnInputConfig{
					{
						Column:     userstore.ResourceID{Name: "rolodex_phone_number"},
						Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					},
				},
				AccessPolicy: userstore.ResourceID{ID: isOperationsAPIKeyAccessPolicy.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: `{id} = ?`,
				},
			},
			idp.IfNotExists(),
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r.createPhoneNumberMutator = createPhoneNumberMutator

	return &r, nil
}

// createContact creates a user record with the specified name, and then executes
// the createPhoneNumber mutator that had previously been configured to add a
// rolodex_phone_number column with the specified consented purposes.

func (r rolodex) createContact(c contact) error {
	userID, err := r.idpClient.CreateUser(r.ctx, userstore.Record{"name": c.name})
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = r.idpClient.ExecuteMutator(
		r.ctx,
		r.createPhoneNumberMutator.ID,
		policy.ClientContext{"api_key": "api_key_for_operations"},
		[]any{userID},
		map[string]idp.ValueAndPurposes{
			"rolodex_phone_number": {
				Value:            c.phoneNumber,
				PurposeAdditions: c.getPhoneNumberPurposeResourceIDs(),
			},
		},
	)
	return ucerr.Wrap(err)
}

// getConsentedPhoneNumberPurposes retrieves the consented purpose names for the
// rolodex_phone_number column for the specified user id.
//
// This is not something a user would typically be recommended to do, but for the
// purpose of this demo, we are showing the consented purposes for the phone number
// column for each created user so that it is clear which phone numbers should be
// returned when calling the phone number accessors that we've configured.

func (r rolodex) getConsentedPhoneNumberPurposes(userID uuid.UUID) (string, error) {
	resp, err := r.idpClient.GetConsentedPurposesForUser(
		r.ctx,
		userID,
		[]userstore.ResourceID{
			{
				Name: "rolodex_phone_number",
			},
		},
	)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if len(resp.Data) != 1 {
		return "", ucerr.Errorf("unexpected number of consented columns: %d", len(resp.Data))
	}

	purposes := ""
	for _, purpose := range resp.Data[0].ConsentedPurposes {
		if purposes != "" {
			purposes = purposes + " "
		}
		purposes = purposes + purpose.Name
	}

	return purposes, nil

}

// listContacts uses the previously configured listContactsAccessor to retrieve the relevant
// columns for each record that have a consented purpose of "operational", and utilizes the
// getConsentedPhoneNumberPurposes method to include the associated consented purposes for
// the rolodex_phone_number column for each user.

func (r rolodex) listContacts() error {
	contacts, err := r.idpClient.ExecuteAccessor(
		r.ctx,
		r.listContactsAccessor.ID,
		policy.ClientContext{"api_key": "api_key_for_operations"},
		[]any{""},
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if len(contacts.Data) == 0 {
		fmt.Printf("\n*** no contacts found - (C)reate some new contacts ***\n")
		return nil
	}

	fmt.Printf("\n%-40s%-40s%-20s%s\n", "ID", "Name", "Phone Number", "Purposes")

	for _, c := range contacts.Data {
		var rec userstore.Record
		if err = json.Unmarshal([]byte(c), &rec); err != nil {
			return ucerr.Wrap(err)
		}
		purposes, err := r.getConsentedPhoneNumberPurposes(rec.UUIDValue("id"))
		if err != nil {
			return ucerr.Wrap(err)
		}

		fmt.Printf("%-40s%-40s%-20s%v\n",
			rec.StringValue("id"),
			rec.StringValue("name"),
			rec.StringValue("rolodex_phone_number"),
			purposes)
	}

	return nil
}

// sendMarketingMessage uses the previously configured getPhoneNumbersForMarketing
// accessor to retrieve all rolodex_phone_numbers that have a consented purpose of
// rolodex_marketing. A client api key is also passed into this method, and must
// also be set to "api_key_for_marketing" in order for the access policy to pass.

func (r rolodex) sendMarketingMessage(message string, clientAPIKey string) error {
	contacts, err := r.idpClient.ExecuteAccessor(
		r.ctx,
		r.getPhoneNumbersForMarketingAccessor.ID,
		policy.ClientContext{"api_key": clientAPIKey},
		[]any{""},
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if len(contacts.Data) == 0 {
		fmt.Printf("\n*** no matching phone numbers with marketing purpose found using client API key '%s' ***\n", clientAPIKey)
		return nil
	}

	fmt.Printf("\nsending marketing message '%s':\n\n", message)

	for _, c := range contacts.Data {
		var rec userstore.Record
		if err = json.Unmarshal([]byte(c), &rec); err != nil {
			return ucerr.Wrap(err)
		}
		fmt.Printf("--> phone number '%s'\n", rec.StringValue("rolodex_phone_number"))
	}
	return nil
}

// teardown cleans up all of the objects that were part of this sample execution before the
// program exits. Since we used a special "rolodex_" prefix for all of these objects, they
// should be disjoint from any previously created objects and thus safe to delete.

func (r *rolodex) teardown() {
	// delete all created users

	contacts, err := r.idpClient.ExecuteAccessor(
		r.ctx,
		r.listContactsAccessor.ID,
		policy.ClientContext{"api_key": "api_key_for_operations"},
		[]any{""},
	)
	if err != nil {
		panic(err)
	}

	for _, c := range contacts.Data {
		var rec userstore.Record
		if err = json.Unmarshal([]byte(c), &rec); err != nil {
			panic(err)
		}
		if err = r.idpClient.DeleteUser(r.ctx, rec.UUIDValue("id")); err != nil {
			panic(err)
		}
	}

	// delete created mutators and accessors

	if err = r.idpClient.DeleteMutator(r.ctx, r.createPhoneNumberMutator.ID); err != nil {
		panic(err)
	}
	if err = r.idpClient.DeleteAccessor(r.ctx, r.getPhoneNumbersForMarketingAccessor.ID); err != nil {
		panic(err)
	}
	if err = r.idpClient.DeleteAccessor(r.ctx, r.listContactsAccessor.ID); err != nil {
		panic(err)
	}

	// delete created access policies and access policy templates

	if err = r.tokenizerClient.DeleteAccessPolicy(r.ctx, r.isMarketingAPIKeyAccessPolicy.ID, r.isMarketingAPIKeyAccessPolicy.Version); err != nil {
		panic(err)
	}
	if err = r.tokenizerClient.DeleteAccessPolicy(r.ctx, r.isOperationsAPIKeyAccessPolicy.ID, r.isOperationsAPIKeyAccessPolicy.Version); err != nil {
		panic(err)
	}
	if err = r.tokenizerClient.DeleteAccessPolicyTemplate(r.ctx, r.isAPIKeyAllowedAccessPolicyTemplate.ID, r.isAPIKeyAllowedAccessPolicyTemplate.Version); err != nil {
		panic(err)
	}

	// delete created purposes

	if err = r.idpClient.DeletePurpose(r.ctx, r.marketingPurpose.ID); err != nil {
		panic(err)
	}
	if err = r.idpClient.DeletePurpose(r.ctx, r.securityPurpose.ID); err != nil {
		panic(err)
	}

	// delete created columns

	if err = r.idpClient.DeleteColumn(r.ctx, r.phoneNumberColumn.ID); err != nil {
		panic(err)
	}
}
