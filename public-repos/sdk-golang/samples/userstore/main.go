package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/gofrs/uuid"
	"github.com/joho/godotenv"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
)

// This sample shows you how to create new columns in the user store and create access policies governing access
// to the data inside those columns. It also shows you how to create, delete and execute accessors and mutators.
// To learn more about these concepts, see docs.userclouds.com.

type sample struct {
	uniqueSuffix                         string
	createdAccessorIDsByName             map[string]uuid.UUID
	createdAccessPolicyIDsByName         map[string]uuid.UUID
	createdAccessPolicyTemplateIDsByName map[string]uuid.UUID
	createdColumnIDsByName               map[string]uuid.UUID
	createdDataTypeIDsByName             map[string]uuid.UUID
	createdMutatorIDsByName              map[string]uuid.UUID
	createdPurposeIDsByName              map[string]uuid.UUID
	createdTokens                        map[string]bool
	createdTransformerIDsByName          map[string]uuid.UUID
	idpClient                            *idp.Client
	tokenizerClient                      *idp.TokenizerClient
}

func newSample() *sample {
	if err := godotenv.Load(); err != nil {
		log.Printf("error loading .env file: %v\n(did you forget to copy `.env.example` to `.env` and substitute values?)", err)
	}

	tenantURL := os.Getenv("USERCLOUDS_TENANT_URL")
	clientID := os.Getenv("USERCLOUDS_CLIENT_ID")
	clientSecret := os.Getenv("USERCLOUDS_CLIENT_SECRET")

	if tenantURL == "" || clientID == "" || clientSecret == "" {
		log.Fatal("missing one or more required environment variables: USERCLOUDS_TENANT_URL, USERCLOUDS_CLIENT_ID, USERCLOUDS_CLIENT_SECRET")
	}

	// reusable token source
	ts, err := jsonclient.ClientCredentialsForURL(tenantURL, clientID, clientSecret, nil)
	if err != nil {
		panic(err)
	}
	// we'll need IDP and tokenizer clients
	idpClient, err := idp.NewClient(tenantURL, idp.JSONClient(ts, jsonclient.StopLogging()))
	if err != nil {
		panic(err)
	}

	return &sample{
		uniqueSuffix:                         fmt.Sprintf("%d", rand.Intn(1000000)),
		createdAccessorIDsByName:             map[string]uuid.UUID{},
		createdAccessPolicyIDsByName:         map[string]uuid.UUID{},
		createdAccessPolicyTemplateIDsByName: map[string]uuid.UUID{},
		createdColumnIDsByName:               map[string]uuid.UUID{},
		createdDataTypeIDsByName:             map[string]uuid.UUID{},
		createdMutatorIDsByName:              map[string]uuid.UUID{},
		createdPurposeIDsByName:              map[string]uuid.UUID{},
		createdTokens:                        map[string]bool{},
		createdTransformerIDsByName:          map[string]uuid.UUID{},
		idpClient:                            idpClient,
		tokenizerClient:                      idpClient.TokenizerClient,
	}
}

// We append a unique value for any non-default userstore entities used in this sample, so that they don't
// conflict with any existing entities already in use in the tenant that the sample is running against, allowing
// us to safely clean up the entities at the end of the sample execution.
func (s sample) getEntityName(name string) string {
	return name + s.uniqueSuffix
}

func (s *sample) setup(ctx context.Context) {
	// create custom us address data type

	addressDT, err := s.idpClient.CreateDataType(
		ctx,
		userstore.ColumnDataType{
			Name:        s.getEntityName("us_address"),
			Description: "a US style address",
			CompositeAttributes: userstore.CompositeAttributes{
				Fields: []userstore.CompositeField{
					{
						Name:     "Street_Address",
						DataType: datatype.String,
					},
					{
						Name:     "City",
						DataType: datatype.String,
					},
					{
						Name:     "State",
						DataType: datatype.String,
					},
					{
						Name:     "Zip",
						DataType: datatype.String,
					},
				},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdDataTypeIDsByName["us_address"] = addressDT.ID

	// create phone number and shipping addresses columns

	phoneNumberC, err := s.idpClient.CreateColumn(
		ctx,
		userstore.Column{
			Name:      s.getEntityName("phone_number"),
			DataType:  datatype.String,
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdColumnIDsByName["phone_number"] = phoneNumberC.ID

	shippingAddressesC, err := s.idpClient.CreateColumn(
		ctx,
		userstore.Column{
			Name:      s.getEntityName("shipping_addresses"),
			DataType:  userstore.ResourceID{ID: addressDT.ID},
			IndexType: userstore.ColumnIndexTypeNone,
			IsArray:   true,
			Constraints: userstore.ColumnConstraints{
				PartialUpdates: true,
				UniqueRequired: true,
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdColumnIDsByName["shipping_addresses"] = shippingAddressesC.ID

	// create access policy template and access policy utilizing that template

	checkAPIKeyAPT, err := s.tokenizerClient.CreateAccessPolicyTemplate(
		ctx,
		policy.AccessPolicyTemplate{
			Name: s.getEntityName("CheckAPIKey"),
			Function: fmt.Sprintf(`function policy(context, params) {
                        return context.client.api_key == params.api_key;
                } // %s`, s.getEntityName("CheckAPIKey")),
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessPolicyTemplateIDsByName["CheckAPIKey"] = checkAPIKeyAPT.ID

	securityAP, err := s.tokenizerClient.CreateAccessPolicy(
		ctx,
		policy.AccessPolicy{
			Name:       s.getEntityName("CheckAPIKeySecurityTeam"),
			PolicyType: policy.PolicyTypeCompositeAnd,
			Components: []policy.AccessPolicyComponent{
				{
					Template:           &userstore.ResourceID{ID: checkAPIKeyAPT.ID},
					TemplateParameters: `{"api_key": "api_key_for_security_team"}`,
				},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessPolicyIDsByName["CheckAPIKeySecurityTeam"] = securityAP.ID

	// Purposes are used to specify the purpose for which a client is requesting access to PII. They are assigned
	// in mutators based on an action of user consent, and they are checked in mutators and accessors to ensure
	// that the client has access to the data. More on accessors and mutators below.

	// "operational" is a system purpose used for general operations of the site that should already exist
	// (so we do not clean it up), but make sure it is there.

	operationalP, err := s.idpClient.CreatePurpose(
		ctx,
		userstore.Purpose{
			Name:        "operational",
			Description: "This purpose is used to grant access to PII for general site operations.",
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}

	// Create a purpose for the security team to access PII for security reasons.

	securityP, err := s.idpClient.CreatePurpose(
		ctx,
		userstore.Purpose{
			Name:        s.getEntityName("security"),
			Description: "This purpose is used to grant access to PII for security reasons.",
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdPurposeIDsByName["security"] = securityP.ID

	// Accessors are configurable APIs that allow a client to retrieve data from the user store. Accessors are
	// intended to be use-case specific. They enforce data usage policies and minimize outbound data from the
	// store for their given use case.

	// Selectors are used to filter the set of users that are returned by an accessor. They are eseentially SQL
	// WHERE clauses and are configured per-accessor / per-mutator referencing column IDs of the userstore.

	passthroughTransformerID := userstore.ResourceID{ID: policy.TransformerPassthrough.ID}

	securityA, err := s.idpClient.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               s.getEntityName("AccessorForSecurity"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			AccessPolicy:       userstore.ResourceID{ID: securityAP.ID},
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: "id"},
					Transformer: passthroughTransformerID,
				},
				{
					Column:      userstore.ResourceID{Name: "updated"},
					Transformer: passthroughTransformerID,
				},
				{
					Column:      userstore.ResourceID{Name: phoneNumberC.Name},
					Transformer: passthroughTransformerID,
				},
				{
					Column:      userstore.ResourceID{Name: shippingAddressesC.Name},
					Transformer: passthroughTransformerID,
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{%s}->>'street_address' LIKE ?", shippingAddressesC.Name),
			},
			Purposes: []userstore.ResourceID{
				{ID: operationalP.ID},
				{ID: securityP.ID},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessorIDsByName["AccessorForSecurity"] = securityA.ID

	// Transformers are used to transform data in the user store. There are 4 different types of transformers:
	// - Passthrough - doesn't modify the data
	// - Transform - transforms the data into a value not resolvable to original value like extracting area code from phone number
	// - TokenizeByValue - transforms the data into a token that is resolvable to the value passed in
	// - TokenizeByReference - transforms the data into a token that is resolvable to the value stored in input column at the time of resolution

	// Define a transformer that will tokenize a phone number into a fixed token (i.e. phone number A always maps to token A)

	// Code that will tokenize phone number into a token which a random sequence of digits in a format of a phone number.

	tokenizePhoneNumberFunction := `function id(len) {
		var s = "0123456789";
		return Array(len).join().split(',').map(function() {
			return s.charAt(Math.floor(Math.random() * s.length));
		}).join('');
	}
	function validate(str) {
		return (str.length === 10);
	}
	function transform(data, params) {
	  // Strip non numeric characters if present
	  orig_data = data;
	  data = data.replace(/\D/g, '');
	  if (data.length === 11 ) {
		data = data.substr(1, 11);
	  }
	  if (!validate(data)) {
			throw new Error('Invalid US Phone Number Provided');
	  }
	  return '1' + id(10);
	}`

	phoneNumberTokenT, err := s.tokenizerClient.CreateTransformer(
		ctx,
		policy.Transformer{
			Name:               s.getEntityName("TokenizePhoneByValReuse"),
			TransformType:      policy.TransformTypeTokenizeByValue,
			ReuseExistingToken: true, // Fixed mapping
			InputDataType:      datatype.String,
			OutputDataType:     datatype.String,
			Function:           fmt.Sprintf("%s // %s", tokenizePhoneNumberFunction, s.getEntityName("TokenizePhoneByValReuse")),
			Parameters:         ``,
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdTransformerIDsByName["TokenizePhoneByValReuse"] = phoneNumberTokenT.ID

	// Define a transformer that will transform a phone number into a country code (i.e. phone number A -> country code)

	phoneNumberT, err := s.tokenizerClient.CreateTransformer(
		ctx,
		policy.Transformer{
			Name:           s.getEntityName("TransformPhoneNumberToCountryName"),
			Description:    "This transformer gets the country name for the phone number",
			TransformType:  policy.TransformTypeTransform,
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			Function: fmt.Sprintf(`function transform(data, params) {
			try {
				return getCountryNameForPhoneNumber(data);
			} catch (e) {
				return "";
			}
		} // %s`, s.getEntityName("TransformPhoneNumberToCountryName")),
			Parameters: ``,
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdTransformerIDsByName["TransformPhoneNumberToCountryName"] = phoneNumberT.ID

	// Define a transformer that will return the zip code of an address

	addressT, err := s.tokenizerClient.CreateTransformer(
		ctx,
		policy.Transformer{
			Name:           s.getEntityName("AddressZipOnly"),
			Description:    "This transformer returns just the zip code of the address.",
			TransformType:  policy.TransformTypeTransform,
			InputDataType:  userstore.ResourceID{ID: addressDT.ID},
			OutputDataType: datatype.String,
			Function: fmt.Sprintf(`function transform(data, params) {
			return data.zip;
		} // %s`, s.getEntityName("AddressZipOnly")),
			Parameters: ``,
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdTransformerIDsByName["AddressZipOnly"] = addressT.ID

	// Define data science accessor with associated access policy

	dataScienceAP, err := s.tokenizerClient.CreateAccessPolicy(
		ctx,
		policy.AccessPolicy{
			Name:       s.getEntityName("CheckAPIKeyDataScienceTeam"),
			PolicyType: policy.PolicyTypeCompositeAnd,
			Components: []policy.AccessPolicyComponent{
				{
					Template:           &userstore.ResourceID{ID: checkAPIKeyAPT.ID},
					TemplateParameters: `{"api_key": "api_key_for_data_science"}`,
				},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessPolicyIDsByName["CheckAPIKeyDataScienceTeam"] = dataScienceAP.ID

	dataScienceA, err := s.idpClient.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               s.getEntityName("AccessorForDataScience"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{ID: phoneNumberC.ID},
					Transformer: userstore.ResourceID{ID: phoneNumberT.ID},
				},
				{
					Column:      userstore.ResourceID{ID: shippingAddressesC.ID},
					Transformer: userstore.ResourceID{ID: addressT.ID},
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{%s} != ?", phoneNumberC.Name),
			},
			Purposes: []userstore.ResourceID{
				{ID: operationalP.ID},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessorIDsByName["AccessorForDataScience"] = dataScienceA.ID

	// Define an accessor that can be used to log user data safely while not exposing user information, using a fixed phone number token to identify the user in logs

	securityLoggingA, err := s.idpClient.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               s.getEntityName("AccessorForSecurityLogging"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			TokenAccessPolicy:  userstore.ResourceID{ID: securityAP.ID},
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: "email"},
					Transformer: userstore.ResourceID{ID: policy.TransformerEmail.ID},
				},
				{
					Column:      userstore.ResourceID{Name: phoneNumberC.Name},
					Transformer: userstore.ResourceID{ID: phoneNumberTokenT.ID},
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ANY(?)"},
			Purposes: []userstore.ResourceID{
				{ID: operationalP.ID},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessorIDsByName["AccessorForSecurityLogging"] = securityLoggingA.ID

	// Mutators are configurable APIs that allow a client to write data to the User Store. Mutators (setters)
	// can be thought of as the complement to accessors (getters).

	// Define a mutator that allows the security team to update phone numbers and shipping addresses

	securityM, err := s.idpClient.CreateMutator(
		ctx,
		userstore.Mutator{
			Name:         s.getEntityName("MutatorForSecurityTeam"),
			AccessPolicy: userstore.ResourceID{ID: securityAP.ID},
			Columns: []userstore.ColumnInputConfig{
				{
					Column:     userstore.ResourceID{Name: phoneNumberC.Name},
					Normalizer: passthroughTransformerID,
				},
				{
					Column:     userstore.ResourceID{Name: shippingAddressesC.Name},
					Normalizer: passthroughTransformerID,
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdMutatorIDsByName["MutatorForSecurityTeam"] = securityM.ID
}

func (s *sample) paginationExample(ctx context.Context) {
	// create a simple accessor to illustrate pagination

	paginationA, err := s.idpClient.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               s.getEntityName("PaginationAccessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: "id"},
					Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
				},
				{
					Column:      userstore.ResourceID{Name: "email"},
					Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ANY(?)"},
			Purposes: []userstore.ResourceID{
				{Name: "operational"},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		panic(err)
	}
	s.createdAccessorIDsByName["PaginationAccessor"] = paginationA.ID

	// create a set of test users

	var userIDs []uuid.UUID
	var userEmails []string

	userEmailPrefix := s.getEntityName("test")
	for i := 0; i < 50; i++ {
		userEmail := fmt.Sprintf("%s_%02d@test.org", userEmailPrefix, i)
		userID, err := s.idpClient.CreateUser(ctx, userstore.Record{"email": userEmail})
		if err != nil {
			panic(err)
		}
		userIDs = append(userIDs, userID)
		userEmails = append(userEmails, userEmail)
	}

	defer func() {
		for _, userID := range userIDs {
			if err := s.idpClient.DeleteUser(ctx, userID); err != nil {
				panic(err)
			}
		}
	}()

	// ascending forward pagination

	s.paginationExecute(ctx, "forward ascending", pagination.CursorBegin, true, true, userIDs...)

	// descending forward pagination

	s.paginationExecute(ctx, "forward descending", pagination.CursorBegin, true, false, userIDs...)

	// ascending backward pagination

	s.paginationExecute(ctx, "backward ascending", pagination.CursorEnd, false, true, userIDs...)

	// descending backward pagination

	s.paginationExecute(ctx, "backward descending", pagination.CursorEnd, false, false, userIDs...)
}

func (s *sample) paginationExecute(
	ctx context.Context,
	description string,
	cursor pagination.Cursor,
	forward bool,
	ascending bool,
	userIDs ...uuid.UUID,
) {
	var options []idp.Option

	if forward {
		options = append(options, idp.Pagination(pagination.StartingAfter(cursor)))
	} else {
		options = append(options, idp.Pagination(pagination.EndingBefore(cursor)))
	}

	if ascending {
		options = append(options, idp.Pagination(pagination.SortOrder(pagination.OrderAscending)))
	} else {
		options = append(options, idp.Pagination(pagination.SortOrder(pagination.OrderDescending)))
	}

	options = append(options, idp.Pagination(pagination.SortKey("email,id")))
	options = append(options, idp.Pagination(pagination.Limit(25)))

	fmt.Printf("\nexecuting %s pagination accessor with cursor '%v':\n", description, cursor)

	resp, err := s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["PaginationAccessor"],
		policy.ClientContext{},
		[]any{userIDs},
		options...,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"- returned %d rows, HasNext = '%v', Next = '%v', HasPrev = '%v', Prev = '%v'\n",
		len(resp.Data),
		resp.HasNext,
		resp.Next,
		resp.HasPrev,
		resp.Prev,
	)

	for _, dataRow := range resp.Data {
		fmt.Printf("-- %s\n", dataRow)
	}

	if forward {
		if resp.HasNext {
			s.paginationExecute(ctx, description, resp.Next, forward, ascending, userIDs...)
		}
	} else {
		if resp.HasPrev {
			s.paginationExecute(ctx, description, resp.Prev, forward, ascending, userIDs...)
		}
	}
}

func (s *sample) example(ctx context.Context) {
	userEmail := fmt.Sprintf("%s@test.org", s.getEntityName("test"))
	uid, err := s.idpClient.CreateUser(ctx, userstore.Record{"email": userEmail})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := s.idpClient.DeleteUser(ctx, uid); err != nil {
			panic(err)
		}
	}()

	// Add the phone number or address using a mutator, but only store with "operational" purpose.

	if _, err := s.idpClient.ExecuteMutator(
		ctx,
		s.createdMutatorIDsByName["MutatorForSecurityTeam"],
		policy.ClientContext{"api_key": "api_key_for_security_team"},
		[]any{uid.String()},
		map[string]idp.ValueAndPurposes{
			s.getEntityName("phone_number"): {
				Value: "14155555555",
				PurposeAdditions: []userstore.ResourceID{
					{Name: "operational"},
				},
			},
			s.getEntityName("shipping_addresses"): {
				ValueAdditions: `[{"state":"IL", "street_address":"742 Evergreen Terrace", "city":"Springfield", "zip":"62704"},
						 {"state":"CA", "street_address":"123 Main St", "city":"Pleasantville", "zip":"94566"}]`,
				PurposeAdditions: []userstore.ResourceID{
					{Name: "operational"},
				},
			},
		}); err != nil {

		panic(err)
	}

	// Use security accessor to get the user details.

	resolved, err := s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["AccessorForSecurity"],
		policy.ClientContext{"api_key": "api_key_for_security_team"},
		[]any{"%Evergreen%"},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("user details for security team are (after first mutator):\n%v\n\n", resolved)

	// Add the phone number or address using a mutator, adding the "security" purpose.

	if _, err := s.idpClient.ExecuteMutator(
		ctx,
		s.createdMutatorIDsByName["MutatorForSecurityTeam"],
		policy.ClientContext{"api_key": "api_key_for_security_team"},
		[]any{uid.String()},
		map[string]idp.ValueAndPurposes{
			s.getEntityName("phone_number"): {
				Value: "14155555555",
				PurposeAdditions: []userstore.ResourceID{
					{Name: s.getEntityName("security")},
				},
			},
			s.getEntityName("shipping_addresses"): {
				ValueAdditions: `[{"state":"IL", "street_address":"742 Evergreen Terrace", "city":"Springfield", "zip":"62704"},
						 {"state":"CA", "street_address":"123 Main St", "city":"Pleasantville", "zip":"94566"}]`,
				PurposeAdditions: []userstore.ResourceID{
					{Name: s.getEntityName("security")},
				},
			},
		}); err != nil {
		panic(err)
	}

	// Use security accessor to get the user details a second time.

	resolved, err = s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["AccessorForSecurity"],
		policy.ClientContext{"api_key": "api_key_for_security_team"},
		[]any{"%Evergreen%"})
	if err != nil {
		panic(err)
	}

	fmt.Printf("user details for security are (after second mutator):\n%v\n\n", resolved)

	// Use data science accessor to get the user details.

	resolved, err = s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["AccessorForDataScience"],
		policy.ClientContext{"api_key": "api_key_for_data_science_team"},
		[]any{"usa"})
	if err != nil {
		panic(err)
	}

	fmt.Printf("user details for data science are:\n%v\n\n", resolved)

	// Use security logging accessor to get the user details for logs emulating two different calls sites.
	// Since the associated phone number transformer is set up to reuse tokens, the same tokens should be
	// returned for both calls.

	resolved, err = s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["AccessorForSecurityLogging"],
		policy.ClientContext{"api_key": "api_key_for_data_science_team"},
		[]any{[]uuid.UUID{uid}})
	if err != nil {
		panic(err)
	}

	fmt.Printf("user details for used for logging at call site 1 are:\n%v\n\n", resolved)

	s.getCreatedTokens(resolved.Data)

	resolved, err = s.idpClient.ExecuteAccessor(
		ctx,
		s.createdAccessorIDsByName["AccessorForSecurityLogging"],
		policy.ClientContext{"api_key": "api_key_for_data_science_team"},
		[]any{[]uuid.UUID{uid}})
	if err != nil {
		panic(err)
	}

	fmt.Printf("user details for used for logging at call site 2:\n%v\n\n", resolved)

	// Resolve the tokenized phone numbers from logs to the original phone numbers for security team

	for _, value := range resolved.Data {
		var m map[string]string
		if err = json.Unmarshal([]byte(value), &m); err != nil {
			panic(err)
		}
		phoneNumberToken := m[s.getEntityName("phone_number")]

		resolvedPhoneNumber, err := s.idpClient.ResolveToken(
			ctx,
			phoneNumberToken,
			policy.ClientContext{"api_key": "api_key_for_security_team"},
			nil)
		if err != nil {
			panic(err)
		}

		fmt.Printf("resolving phone number token %s to phone number %s\n", phoneNumberToken, resolvedPhoneNumber)
	}

	s.getCreatedTokens(resolved.Data)
}

func (s *sample) getCreatedTokens(responses []string) {
	for _, response := range responses {
		var m map[string]string
		if err := json.Unmarshal([]byte(response), &m); err != nil {
			panic(err)
		}
		if emailToken, found := m["email"]; found {
			s.createdTokens[emailToken] = true
		}

		if phoneNumberToken, found := m[s.getEntityName("phone_number")]; found {
			s.createdTokens[phoneNumberToken] = true
		}
	}
}

func (s *sample) cleanup(ctx context.Context) {
	for token := range s.createdTokens {
		if err := s.tokenizerClient.DeleteToken(ctx, token); err != nil {
			panic(err)
		}
	}
	s.createdTokens = map[string]bool{}

	for _, mid := range s.createdMutatorIDsByName {
		if err := s.idpClient.DeleteMutator(ctx, mid); err != nil {
			panic(err)
		}
	}
	s.createdMutatorIDsByName = map[string]uuid.UUID{}

	for _, aid := range s.createdAccessorIDsByName {
		if err := s.idpClient.DeleteAccessor(ctx, aid); err != nil {
			panic(err)
		}
	}
	s.createdAccessorIDsByName = map[string]uuid.UUID{}

	for _, tid := range s.createdTransformerIDsByName {
		if err := s.tokenizerClient.DeleteTransformer(ctx, tid); err != nil {
			panic(err)
		}
	}
	s.createdTransformerIDsByName = map[string]uuid.UUID{}

	for _, apid := range s.createdAccessPolicyIDsByName {
		if err := s.tokenizerClient.DeleteAccessPolicy(ctx, apid, 0); err != nil {
			panic(err)
		}
	}
	s.createdAccessPolicyIDsByName = map[string]uuid.UUID{}

	for _, aptid := range s.createdAccessPolicyTemplateIDsByName {
		if err := s.tokenizerClient.DeleteAccessPolicyTemplate(ctx, aptid, 0); err != nil {
			panic(err)
		}
	}
	s.createdAccessPolicyTemplateIDsByName = map[string]uuid.UUID{}

	for _, cid := range s.createdColumnIDsByName {
		if err := s.idpClient.DeleteColumn(ctx, cid); err != nil {
			panic(err)
		}
	}
	s.createdColumnIDsByName = map[string]uuid.UUID{}

	for _, pid := range s.createdPurposeIDsByName {
		if err := s.idpClient.DeletePurpose(ctx, pid); err != nil {
			panic(err)
		}
	}
	s.createdPurposeIDsByName = map[string]uuid.UUID{}

	for _, id := range s.createdDataTypeIDsByName {
		if err := s.idpClient.DeleteDataType(ctx, id); err != nil {
			panic(err)
		}
	}
	s.createdDataTypeIDsByName = map[string]uuid.UUID{}
}

func main() {
	ctx := context.Background()

	s := newSample()
	s.setup(ctx)
	s.example(ctx)
	s.paginationExample(ctx)
	s.cleanup(ctx)
}
