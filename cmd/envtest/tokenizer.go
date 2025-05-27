package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/async"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/uclog"
)

const (
	poolSizeTokenizer    int = 100
	commonPoolResolveOps int = 10
)

func perThreadAccessPolicyName(threadNum int) string {
	return fmt.Sprintf("EnvTestAccessPolicyForSecurity_%d", threadNum)
}

func perThreadTransformerName(threadNum int) string {
	return fmt.Sprintf("EnvTestTransformerForSecurity_%d", threadNum)
}

func perThreadSampleAccessPolicyTemplateName(threadNum int) string {
	return fmt.Sprintf("EnvTestAccessPolicyTemplateTokenizerUser_%d", threadNum)
}

func perThreadSampleAccessPolicyName(threadNum int) string {
	return fmt.Sprintf("EnvTestAccessPolicyTokenizerUser_%d", threadNum)
}

func perThreadTokenContents(threadNum int) string {
	return fmt.Sprintf("Token_%d", threadNum)
}

func perThreadSampleTokenContents(threadNum int) string {
	return fmt.Sprintf("Hello World_%d", threadNum)
}

func cleanupTokenizer(ctx context.Context, tc *idp.TokenizerClient) {
	startTime := time.Now().UTC()
	uclog.Debugf(ctx, "Tokenizer: Starting per test cleanup")

	deleteCount := 0
	aPolicies, err := tc.ListAccessPolicies(ctx, false)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: ListAccessPolicies: %v", err)
	} else {
		uclog.Debugf(ctx, "Tokenizer: Found %d access policies in the tenant", len(aPolicies.Data))
		accessPolicyNames := make(map[string]bool)
		for i := range envTestCfg.Threads {
			accessPolicyNames[perThreadAccessPolicyName(i)] = true
			accessPolicyNames[perThreadSampleAccessPolicyName(i)] = true
		}

		for _, ap := range aPolicies.Data {
			if accessPolicyNames[ap.Name] {
				uclog.Debugf(ctx, "Tokenizer: Deleting %s access policy", ap.Name)

				// First clear any tokens that might be left
				for i := range envTestCfg.Threads {
					tokens, err := tc.LookupTokens(ctx, perThreadTokenContents(i), userstore.ResourceID{Name: perThreadTransformerName(i)}, userstore.ResourceID{ID: ap.ID})
					if err != nil {
						logErrorf(ctx, err, "Tokenizer: Cleanup LookupTokens failed: %v", err)
					}
					tokensSample, err := tc.LookupTokens(ctx, perThreadSampleTokenContents(i), userstore.ResourceID{Name: perThreadTransformerName(i)}, userstore.ResourceID{ID: ap.ID})
					if err != nil {
						logErrorf(ctx, err, "Tokenizer: Cleanup LookupTokens failed: %v", err)
					}

					tokens = append(tokens, tokensSample...)
					for _, token := range tokens {
						uclog.Debugf(ctx, "Tokenizer: Deleting token %s referencing %s policy", token, ap.Name)
						if err := tc.DeleteToken(ctx, token); err != nil {
							logErrorf(ctx, err, "DeleteToken for %v ap %v failed: %v", token, ap.ID, err)
						}
					}
				}

				if err := tc.DeleteAccessPolicy(ctx, ap.ID, 0); err != nil {
					logErrorf(ctx, err, "Tokenizer: DeleteAccessPolicy for ap %v failed during cleanup: %v", ap.ID, err)
				}
				deleteCount++
			}
		}
	}

	transformerNames := make(map[string]bool)
	for i := range envTestCfg.Threads {
		transformerNames[perThreadTransformerName(i)] = true
	}

	gPolicies, err := tc.ListTransformers(ctx)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: ListAccessPolicies: %v", err)
	} else {
		uclog.Debugf(ctx, "Tokenizer: Found %d transformers in the tenant", len(gPolicies.Data))
		for _, gp := range gPolicies.Data {
			if transformerNames[gp.Name] {
				uclog.Debugf(ctx, "Tokenizer: Deleting %s transformers", gp.Name)
				if err := tc.DeleteTransformer(ctx, gp.ID); err != nil {
					logErrorf(ctx, err, "Tokenizer: DeleteTransformer for ap %v failed: %v", gp.ID, err)
				}
				deleteCount++
			}
		}
	}

	wallTime := time.Now().UTC().Sub(startTime)
	uclog.Debugf(ctx, "Tokenizer: Finished clean up. Deleted %d policies in %v", deleteCount, wallTime)
}

func tokenizerBuildInLoad(ctx context.Context, threadNum int, tc *idp.TokenizerClient) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "Tokenizer: Starting build in policies test for thread %d", threadNum)

	// Create a tokens using each of build in policies and verify that it resolves
	aPolicies, err := tc.ListAccessPolicies(ctx, false)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d ListAccessPolicies: %v", threadNum, err)
	}

	transformers, err := tc.ListTransformers(ctx)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d ListTransformers: %v", threadNum, err)
	}

	// TODO - implement attributes on policies so we can only do this for system policies
	for _, ap := range aPolicies.Data {
		if ap.ID != policy.AccessPolicyAllowAll.ID {
			continue
		}
		// Right now we only provision single access policy that doesn't require context
		var resolutionContext policy.ClientContext
		for _, t := range transformers.Data {

			tdata := ""
			switch t.ID {
			case policy.TransformerCreditCard.ID:
				tdata = "1234-5678-1234-5678"
			case policy.TransformerFullName.ID:
				tdata = "John Smith"
			case policy.TransformerSSN.ID:
				tdata = "300-99-8888"
			case policy.TransformerEmail.ID:
				tdata = "userclouds@gmail.com"
			case policy.TransformerUUID.ID:
				tdata = fmt.Sprintf("Data UUDI %d", rand.Int())
			case policy.TransformerPassthrough.ID:
				tdata = fmt.Sprintf("Data Passthrough %v", uuid.Must(uuid.NewV4()))
			default:
				continue
			}

			if t.TransformType != policy.TransformTypeTokenizeByValue {
				transformer := &policy.Transformer{
					Name:           t.Name + "tokenize",
					Description:    t.Description,
					InputDataType:  t.InputDataType,
					OutputDataType: t.OutputDataType,
					TransformType:  policy.TransformTypeTokenizeByValue,
					Function:       t.Function,
					Parameters:     t.Parameters,
				}
				transformer, err = tc.CreateTransformer(ctx, *transformer, idp.IfNotExists())
				if err != nil {
					logErrorf(ctx, err, "Tokenizer : Thread %2d CreateTransformer for ap %v, gp %v failed: %v", threadNum, ap.ID, t.ID, err)
					continue
				}

				t = *transformer
			}

			if err != nil {
				logErrorf(ctx, err, "Tokenizer : Thread %2d CreateTransformer for ap %v, gp %v failed: %v", threadNum, ap.ID, t.ID, err)
			}

			token, err := tc.CreateToken(ctx, tdata, userstore.ResourceID{ID: t.ID}, userstore.ResourceID{ID: ap.ID})
			if err != nil {
				logErrorf(ctx, err, "Tokenizer : Thread %2d CreateToken for ap %v, gp %v failed: %v", threadNum, ap.ID, t.ID, err)
			} else {
				// Only try to resolve if creation was successful
				got, err := tc.ResolveToken(ctx, token, resolutionContext, nil)
				if err != nil {
					logErrorf(ctx, err, "Tokenizer : Thread %2d ResolveToken for ap %v, gp %v failed: %v", threadNum, ap.ID, t.ID, err)
				} else if got != tdata {
					logErrorf(ctx, err, "Tokenizer : Thread %2d ResolveToken for ap %v, gp %v got the wrong data back expected %v got %v", threadNum, ap.ID, t.ID, tdata, got)
				}
			}

			if err := tc.DeleteToken(ctx, token); err != nil {
				logErrorf(ctx, err, "DeleteToken for ap %v, gp %v failed: %v", ap.ID, t.ID, err)
			}
		}
	}
	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "Tokenizer: Finished buildin policies test for thread %d in %v", threadNum, wallTime)
}

func tokenizerPerThreadLoad(ctx context.Context, threadNum int, tc *idp.TokenizerClient) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "Tokenizer: Starting per thread objects test for thread %d", threadNum)
	apt := &policy.AccessPolicyTemplate{
		Name:        "APIKeyTemplate",
		Description: "Checks API Key",
		Function:    "function policy(context, params) { return context.client.api_key === params.api_key; }",
	}

	aptID := uuid.Nil
	apt, err := tc.CreateAccessPolicyTemplate(ctx, *apt, idp.IfNotExists())
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateAccessPolicyTemplate failed: %v", threadNum, err)
	} else {
		aptID = apt.ID
	}

	// Create new access and transformers and use them to get and evaluate tokens
	ap := &policy.AccessPolicy{
		Name: perThreadAccessPolicyName(threadNum),
		Components: []policy.AccessPolicyComponent{{
			Template:           &userstore.ResourceID{ID: aptID},
			TemplateParameters: `{"api_key": "api_key_security"}`}},
		PolicyType: policy.PolicyTypeCompositeAnd,
	}

	// Use apID/tID constant to not crash when calls fail
	apID := uuid.Nil
	ap, err = tc.CreateAccessPolicy(ctx, *ap)
	if err != nil {
		// TODO DEVEXP when policy already exists we should return the id of conflicting policy
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateAccessPolicy failed: %v", threadNum, err)
	} else {
		apID = ap.ID
	}

	t := &policy.Transformer{
		Name:           perThreadTransformerName(threadNum),
		Description:    fmt.Sprintf(`Thread %d transformer`, threadNum),
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTokenizeByValue,
		Function: fmt.Sprintf(`function transform(data, params) {
				return data + params.Append;
				} //%v`, threadNum),
		Parameters: `{
					"Append": "_type_custom"
				}`,
	}
	tID := uuid.Nil
	t, err = tc.CreateTransformer(ctx, *t)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateTransformer failed: %v", threadNum, err)
	} else {
		tID = t.ID
	}

	// Read the created policies back from the server
	ap, err = tc.GetAccessPolicy(ctx, userstore.ResourceID{ID: apID})
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d GetAccessPolicy failed: %v", threadNum, err)
	} else {
		if ap.ID != apID || ap.Version != 0 {
			logErrorf(ctx, err, "Tokenizer: Thread %2d GetAccessPolicy returned unexpected value: %v", threadNum, ap)
		}
	}

	t, err = tc.GetTransformer(ctx, userstore.ResourceID{ID: tID})
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d GetTransformer failed: %v", threadNum, err)
	} else {
		if t.ID != tID {
			logErrorf(ctx, err, "Tokenizer: Thread %2d GetTransformer returned unexpected value: %v", threadNum, t)
		}
	}

	tdata := perThreadTokenContents(threadNum) // Use fixed token content to make sure we can look it up during cleanup

	token, err := tc.CreateToken(ctx, tdata, userstore.ResourceID{ID: tID}, userstore.ResourceID{ID: apID})
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateToken for ap %v, gp %v failed: %v", threadNum, apID, tID, err)
	} else {
		// Only try to resolve if creation was successful
		if token != tdata+"_type_custom" {
			logErrorf(ctx, err, "Tokenizer: Thread %2d Token has unexpected value %v expect %v ", threadNum, token, tdata+"_type_custom")
		}
		got, err := tc.ResolveToken(ctx, token, policy.ClientContext{"api_key": "api_key_security"}, nil)
		if err != nil {
			logErrorf(ctx, err, "Tokenizer: Thread %2d ResolveToken for ap %v, gp %v failed: %v", threadNum, apID, tID, err)

		} else if got != tdata {
			logErrorf(ctx, err, "Tokenizer : Thread %2d ResolveToken for ap %v, gp %v got the wrong data back expected %v got %v", threadNum, apID, tID, tdata, got)
		}

		// Should fail with a different api key
		got, err = tc.ResolveToken(ctx, token, policy.ClientContext{"api_key": "api_key_marketing"}, nil)
		// TODO DEXEXP - we should probably error in this case but figure out how to keep it consistent with multi resolve
		if err != nil || got != "" {
			logErrorf(ctx, err, "Tokenizer: Thread %2d Resolve token didn't fail as expected for token %v", threadNum, token)
		}

	}

	if err := tc.DeleteToken(ctx, token); err != nil {
		logErrorf(ctx, err, "DeleteToken for ap %v, gp %v failed: %v", apID, tID, err)
	}

	if err := tc.DeleteAccessPolicy(ctx, apID, 0); err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d DeleteAccessPolicy2 for ap %v failed: %v", threadNum, apID, err)
	}

	if err := tc.DeleteTransformer(ctx, tID); err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d DeleteTransformer for ap %v failed: %v", threadNum, tID, err)
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "Tokenizer: Finished per thread objects test for thread %d in %v", threadNum, wallTime)
}

func tokenizerCommonPool(ctx context.Context, threadNum int, tc *idp.TokenizerClient, tokens []string, data []string) {
	// Resolve some tokens from the common pool
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "Tokenizer: Starting common pool test for thread %d", threadNum)
	for range commonPoolResolveOps {
		ti := rand.Intn(len(tokens))
		uclog.Debugf(ctx, "Tokenizer: Thread %2d testing common pool token index %3d, token %v, val %v", threadNum, ti, tokens[ti], data[ti])
		got, err := tc.ResolveToken(ctx, tokens[ti], nil, nil)
		if err != nil {
			logErrorf(ctx, err, "Tokenizer: Thread %2d ResolveToken for common pool token %v failed: %v", threadNum, tokens[ti], err)
		} else if got != data[ti] {
			logErrorf(ctx, err, "Tokenizer: Thread %2d ResolveTokenfor for common pool token %v  got the wrong data back expected %v got %v", threadNum, tokens[ti], data[ti], got)
		}
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "Tokenizer: Finished common pool for thread %d with %d ops in %v", threadNum, commonPoolResolveOps, wallTime)
}

func tokenizerSampleLoad(ctx context.Context, threadNum int, tc *idp.TokenizerClient) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "Tokenizer: Starting sample test for thread %d", threadNum)
	// Run our sample approx code
	recipient := "john"
	policyTemplateName := perThreadSampleAccessPolicyTemplateName(threadNum)
	policyName := perThreadSampleAccessPolicyName(threadNum)

	templateDef := policy.AccessPolicyTemplate{
		Name:     policyTemplateName,
		Function: fmt.Sprintf(`function policy(context, params) { if (context.client.user === params.recipient) { return true; } return false; } //%v`, threadNum),
	}
	uclog.Debugf(ctx, "Tokenizer: Thread %2d Creating template %s on thread %d", threadNum, policyTemplateName, threadNum)
	templateID := uuid.Nil
	template, err := tc.CreateAccessPolicyTemplate(ctx, templateDef, idp.IfNotExists())
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateTemplate Sample %v failed: %v", threadNum, templateDef, err)
	} else {
		if template.Name != policyTemplateName {
			logErrorf(ctx, err, "Tokenizer: Thread %2d CreateTemplate sample returned unexpected value: %v", threadNum, template)
		}
		templateID = template.ID
	}

	uclog.Debugf(ctx, "Tokenizer: Thread %2d Creating policy %s on thread %d", threadNum, policyName, threadNum)
	policyDef := policy.AccessPolicy{
		Name:       policyName,
		Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: templateID}, TemplateParameters: fmt.Sprintf(`{"recipient": "%s"}`, recipient)}},
		PolicyType: policy.PolicyTypeCompositeAnd,
	}

	ap, err := tc.CreateAccessPolicy(ctx, policyDef)
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateAccessPolicy Sample %v failed: %v", threadNum, policyDef, err)
		return // can't continue if this create failed
	}

	if ap.Name != policyName || ap.Version != 0 {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateAccessPolicy sample returned unexpected value: %v", threadNum, ap)
	}
	apID := ap.ID

	message := perThreadSampleTokenContents(threadNum) // Use fixed token content to make sure we can look it up during cleanup
	token, err := tc.CreateToken(ctx, message, userstore.ResourceID{ID: policy.TransformerUUID.ID}, userstore.ResourceID{ID: apID})
	if err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d CreateToken Sample for ap %v  failed: %v", threadNum, apID, err)
		return // can't continue if this create failed
	}

	viewer := "john"
	resolved, err := tc.ResolveToken(ctx, token, policy.ClientContext{"user": viewer}, nil)
	if err != nil || resolved != message {
		logErrorf(ctx, err, "Tokenizer: Thread %2d ResolveToken for Sample ap %v failed: %v message: %v expected : %v", threadNum, apID, err, resolved, message)
	}
	viewer = "bob"
	resolved, err = tc.ResolveToken(ctx, token, policy.ClientContext{"user": viewer}, nil)
	if err != nil || resolved != "" {
		logErrorf(ctx, err, "Tokenizer: Thread %2d Resolve token didn't fail as expected for token %v", threadNum, token)
	}

	if err := tc.DeleteToken(ctx, token); err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d DeleteToken for ap %v, transformer %s failed: %v", threadNum, apID, policy.TransformerUUID.ID, err)
	}

	if err := tc.DeleteAccessPolicy(ctx, apID, 0); err != nil {
		logErrorf(ctx, err, "Tokenizer: Thread %2d DeleteAccessPolicy for ap %v failed: %v", threadNum, apID, err)
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "Tokenizer: Finished sample test for thread %d in %v", threadNum, wallTime)
}

func tokenizerTestWorker(ctx context.Context, threadNum int, tc *idp.TokenizerClient, tokens []string, data []string, numOps int) {
	for range numOps {
		// Create tokens using the build in policies
		tokenizerBuildInLoad(ctx, threadNum, tc)
		// Do some operations on the per thread objects
		tokenizerPerThreadLoad(ctx, threadNum, tc)
		// Resolve some tokens from the common pool
		tokenizerCommonPool(ctx, threadNum, tc, tokens, data)
		// Replicate the sample code
		tokenizerSampleLoad(ctx, threadNum, tc)
	}
}

func tokenizerTest(ctx context.Context, tenantURL string, tokenSource jsonclient.Option, iterations int) {
	for i := 1; i < iterations+1; i++ {
		// TODO DEVEXP this should return an error
		tc := idp.NewTokenizerClient(tenantURL, idp.JSONClient(tokenSource))
		testStart := time.Now().UTC()
		uclog.Infof(ctx, "Tokenizer: Starting an environment test for %d worker threads - %v (%d/%d)", envTestCfg.Threads, tenantURL, i, iterations)

		// Clean in case there are policies left from previous test run
		cleanupTokenizer(ctx, tc)

		// Create common set of tokens to be resolved across multiple worker threads
		var tokens = make([]string, poolSizeTokenizer)
		var data = make([]string, poolSizeTokenizer)
		for i := range tokens {
			var err error
			data[i] = fmt.Sprintf("Data %d", rand.Int())
			tokens[i], err = tc.CreateToken(ctx, data[i], userstore.ResourceID{ID: policy.TransformerUUID.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
			if err != nil {
				logErrorf(ctx, err, "Tokenizer: CreateToken failed for common pool: %v", err)
				return
			}
		}

		wg := sync.WaitGroup{}
		for i := range envTestCfg.Threads {
			wg.Add(1)
			threadNum := i
			async.Execute(func() {
				tokenizerTestWorker(ctx, threadNum, tc, tokens, data, defaultThreadOpCount)
				wg.Done()
			})
		}
		wg.Wait()

		for _, t := range tokens {
			if err := tc.DeleteToken(ctx, t); err != nil {
				logErrorf(ctx, err, "Tokenizer: DeleteToken failed for common pool: %v", err)
			}
		}

		wallTime := time.Now().UTC().Sub(testStart)
		uclog.Infof(ctx, "Tokenizer: Completed tokenizer environment test for %d worker threads in %v (%d/%d)", envTestCfg.Threads, wallTime, i, iterations)
	}
}
