package tokenizer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"userclouds.com/authz"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/ucv8go"
)

type accessPolicyTemplateExecutor struct {
	authzClient    *authz.Client
	jsContext      *ucv8go.Context
	consoleBuilder *strings.Builder
	s              *storage.Storage
}

func newAccessPolicyTemplateExecutor(
	authzClient *authz.Client,
	s *storage.Storage,
) (*accessPolicyTemplateExecutor, error) {
	return &accessPolicyTemplateExecutor{
		authzClient: authzClient,
		s:           s,
	}, nil
}

func (apte *accessPolicyTemplateExecutor) cleanup() {
	if apte.jsContext != nil {
		res := apte.jsContext.Res
		apte.jsContext.Close()
		res.Release()
		apte.jsContext = nil
	}
}

func (apte *accessPolicyTemplateExecutor) execute(
	ctx context.Context,
	template *storage.AccessPolicyTemplate,
	apContext policy.AccessPolicyContext,
	contextString string,
	params string,
) (bool, error) {
	// Request may contain PII so don't log it in prod
	uclog.DebugfPII(ctx, "executing access policy template: %+v", template.ID)

	start := time.Now().UTC()
	defer logAPTemplateDuration(ctx, template.ID, template.Version, start)
	logAPTemplateCall(ctx, template.ID, template.Version)

	if err := template.Validate(); err != nil {
		return false, ucerr.Wrap(err)
	}

	allowed := false

	// Check if we can execute this policy natively
	if nt, isNativeTemplate := nativeTemplates[template.ID][template.Version]; isNativeTemplate {
		allowed = nt.fun(ctx, apte.authzClient, apContext, params)
	} else {
		jsContext, err := apte.getJSContext(ctx)
		if err != nil {
			return false, ucerr.Wrap(err)
		}

		// Function must include a JS function named policy that takes (context, params) and returns bool | error
		// TODO: include "system context" like IP address
		// newline prevents trailing comments from breaking the function
		jsScript := fmt.Sprintf("%s;\npolicy(%v, %v);", template.Function, contextString, params)

		val, err := jsContext.RunScript(jsScript, fmt.Sprintf("%v.js", template.ID))
		if err != nil {
			uclog.Warningf(ctx, "failed to execute access policy template %v v%v (%s): %v", template.ID, template.Version, jsScript, err)
			logAPTemplateError(ctx, template.ID, template.Version)
			wp := jsonclient.Error{
				StatusCode: http.StatusBadRequest,
				Body:       fmt.Sprintf("error executing access policy template: %v", err.Error()),
			}
			return false, ucerr.Wrap(wp)

		}
		uclog.Verbosef(ctx, "result: %+v", val)

		allowed = val.Boolean()
	}

	logAPTemplateResult(ctx, template.ID, template.Version, allowed)

	return allowed, nil
}

func (apte *accessPolicyTemplateExecutor) getConsoleOutput() string {
	if apte.consoleBuilder == nil {
		return ""
	}
	return apte.consoleBuilder.String()
}

func (apte *accessPolicyTemplateExecutor) getJSContext(ctx context.Context) (*ucv8go.Context, error) {
	if apte.jsContext == nil {
		jsContext, cb, err := ucv8go.NewJSContext(ctx, apte.authzClient, auditlog.AccessPolicyCustom, newPolicySecretResolver(apte.s))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		apte.jsContext = jsContext
		apte.consoleBuilder = cb
	}

	return apte.jsContext, nil
}
