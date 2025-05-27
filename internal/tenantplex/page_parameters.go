package tenantplex

import (
	"fmt"
	"strings"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/telephony"
	pageparams "userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	paramtype "userclouds.com/internal/pageparameters/parametertype"
)

func (tenant *TenantConfig) classifyMFAMethods() (disabledMFAMethods string, enabledMFAMethods string) {
	var disabledBuilder strings.Builder
	var enabledBuilder strings.Builder

	foundRecoveryCode := false
	for mt := range strings.SplitSeq(paramtype.MFAMethodTypes, ",") {
		enabled := false
		switch oidc.MFAChannelType(mt) {
		case oidc.MFAAuthenticatorChannel:
			enabled = true
		case oidc.MFAEmailChannel:
			// a valid TenantConfig will always have a valid email provider
			enabled = true
		case oidc.MFASMSChannel:
			enabled = telephony.IsConfigured(&tenant.PlexMap.TelephonyProvider)
		case oidc.MFARecoveryCodeChannel:
			foundRecoveryCode = true
			continue
		}

		if enabled {
			if enabledBuilder.Len() != 0 {
				enabledBuilder.WriteString(",")
			}
			enabledBuilder.WriteString(mt)
		} else {
			if disabledBuilder.Len() != 0 {
				disabledBuilder.WriteString(",")
			}
			disabledBuilder.WriteString(mt)
		}
	}

	// we allow recovery codes if there is at least one other enabled MFA method

	if foundRecoveryCode {
		if enabledBuilder.Len() != 0 {
			enabledBuilder.WriteString(",")
			enabledBuilder.WriteString(oidc.MFARecoveryCodeChannel.String())
		} else {
			if disabledBuilder.Len() != 0 {
				disabledBuilder.WriteString(",")
			}
			disabledBuilder.WriteString(oidc.MFARecoveryCodeChannel.String())
		}
	}

	return disabledBuilder.String(), enabledBuilder.String()
}

func (op *OIDCProviders) classifyProviders(defaultDisabled string, defaultEnabled string) (disabledMethods string, enabledMethods string, oidcSettings string) {
	var disabledBuilder strings.Builder
	var enabledBuilder strings.Builder
	var settingsBuilder strings.Builder

	disabledBuilder.WriteString(defaultDisabled)
	enabledBuilder.WriteString(defaultEnabled)

	for _, p := range op.Providers {
		provider, err := oidc.GetProvider(&p)
		if err != nil {
			// in practice, this should never happen
			continue
		}

		if settingsBuilder.Len() != 0 {
			settingsBuilder.WriteString(",")
		}
		settingsBuilder.WriteString(
			fmt.Sprintf("%s:%s:%s:%s",
				provider.GetName(),
				provider.GetDescription(),
				provider.GetLoginButtonDescription(),
				provider.GetMergeButtonDescription()))

		if provider.IsConfigured() {
			if enabledBuilder.Len() != 0 {
				enabledBuilder.WriteString(",")
			}
			enabledBuilder.WriteString(provider.GetName())
		} else {
			if disabledBuilder.Len() != 0 {
				disabledBuilder.WriteString(",")
			}
			disabledBuilder.WriteString(provider.GetName())
		}
	}

	return disabledBuilder.String(), enabledBuilder.String(), settingsBuilder.String()
}

// GetAuthenticationMethods returns a slice of the selected authentication methods
// for a tenant and app
func GetAuthenticationMethods(tenant *TenantConfig, app *App) ([]string, error) {
	pg := MakeRenderParameterGetter(tenant, app)
	cd := MakeParameterClientData(tenant, app)
	p, found := pg(pagetype.EveryPage, param.AuthenticationMethods)
	if !found {
		return nil, ucerr.New("could not find any authentication methods")
	}
	p, err := p.ApplyClientData(cd)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return paramtype.GetOptions(p.Value), nil
}

const defaultDisabledAuthMethods = ""
const defaultEnabledAuthMethods = "password,passwordless"

// MakeParameterClientData populates a ClientData based on the settings of the passed in tenant
// and app, to be used for replacing any template parameters referenced in page parameters
func MakeParameterClientData(tenant *TenantConfig, app *App) param.ClientData {
	disabledAuthMethods, enabledAuthMethods, oidcAuthSettings := tenant.OIDCProviders.classifyProviders(defaultDisabledAuthMethods, defaultEnabledAuthMethods)
	disabledMFAMethods, enabledMFAMethods := tenant.classifyMFAMethods()
	data := param.ClientData{
		AllowCreation:                 "true",
		AppName:                       app.Name,
		DisabledAuthenticationMethods: disabledAuthMethods,
		DisabledMFAMethods:            disabledMFAMethods,
		EnabledAuthenticationMethods:  enabledAuthMethods,
		EnabledMFAMethods:             enabledMFAMethods,
		OIDCAuthenticationSettings:    oidcAuthSettings,
		PasswordResetEnabled:          "true",
	}
	if tenant.DisableSignUps {
		data.AllowCreation = "false"
	}

	return data
}

// MakeParameterRetrievalTools returns a list of parameter getters ordered from least specific
// (defaults) to most specific (customizations set for an app), and a ClientData instance for
// the tenant and app
func MakeParameterRetrievalTools(tenant *TenantConfig, app *App) ([]pageparams.ParameterGetter, param.ClientData) {
	return []pageparams.ParameterGetter{
			pagetype.DefaultParameterGetter,
			func(pt pagetype.Type, pn param.Name) (param.Parameter, bool) {
				p, found := tenant.PageParameters[pt][pn]
				return p, found
			},
			func(pt pagetype.Type, pn param.Name) (param.Parameter, bool) {
				p, found := app.PageParameters[pt][pn]
				return p, found
			},
		},
		MakeParameterClientData(tenant, app)
}

// MakeRenderParameterGetter a parameter getter that will return the most specific value
// for a page type and parameter, for either the specified page type, or all pages if
// the parameter does not apply for the page type
func MakeRenderParameterGetter(tenant *TenantConfig, app *App) pageparams.ParameterGetter {
	getter := func(pt pagetype.Type, pn param.Name) (param.Parameter, bool) {
		if p, found := app.PageParameters[pt][pn]; found {
			return p, true
		}
		if p, found := tenant.PageParameters[pt][pn]; found {
			return p, true
		}
		return pagetype.DefaultParameterGetter(pt, pn)
	}
	return func(pt pagetype.Type, pn param.Name) (param.Parameter, bool) {
		if p, found := getter(pt, pn); found {
			return p, true
		}
		return getter(pagetype.EveryPage, pn)
	}
}
