package featureflags

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/gofrs/uuid"
	statsig "github.com/statsig-io/go-sdk"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
)

// Config is the configuration for the feature flags package
type Config struct {
	APIKey         secret.String `yaml:"api_key" json:"api_key" validate:"notempty"`
	VerboseLogging bool          `yaml:"verbose" json:"verbose"`
}

var notInitWarningLogged = false
var initLock = sync.RWMutex{}

func init() {
	if !universe.Current().IsTestOrCI() {
		notInitWarningLogged = false
		return
	}
	initForTests()
}

//go:generate genvalidate Config

// Flag is a feature flag name as string as it appears in the statsig console
type Flag string

// DynamicConfig is a dynamic config name as string as it appears in the statsig console
type DynamicConfig string

type flagScope string

const (
	// OnMachineEdgesCacheDisable is the flag for turning off one machine edges tenant cache in authz
	OnMachineEdgesCacheDisable Flag = "edges-tenant-cache-disable"
	// OnMachineEdgesCacheVerify is the flag for turning off extra verification in one machine edges cache
	OnMachineEdgesCacheVerify Flag = "edges-tenant-cache-verify"
	// OnMachineEdgesCacheCompareResults is the flag for executing checkattribute side by side with cache and without cache
	OnMachineEdgesCacheCompareResults Flag = "edges-tenant-cache-compare-results"
	// GlobalAccessPolicies is the flag for turning on/off global access policies feature
	GlobalAccessPolicies Flag = "global-access-policies"
	// AsyncAuthzForUserStore makes authz calls on user creation async
	AsyncAuthzForUserStore Flag = "async-authz-for-userstore"
	// ReadFromReadReplica is the flag for reading from read replica
	ReadFromReadReplica Flag = "read-from-read-replica"
	// CheckAttributeViaService is the flag for checking attributes via dedicated service
	CheckAttributeViaService Flag = "check-attribute-via-service"

	// DisabledWorkerTasks is the flag for disabling worker tasks
	DisabledWorkerTasks DynamicConfig = "disabled-worker-tasks"

	userIDSchemeVersion           = "v1"
	scopeTenant         flagScope = "tenant"
	scopeCompany        flagScope = "company"
	scopeHostname       flagScope = "hostname"
	scopeGlobal         flagScope = "global"
)
const featureflagsSubsystem = ucmetrics.Subsystem("featureflags")

var (
	failedEventUpload = ucmetrics.CreateCounter(featureflagsSubsystem, "failed_event_upload", "Number of times event upload failed")

	logEventFailureExpression = regexp.MustCompile(`Failed to log \d+ events`)
)

func getLogCallback(ctx context.Context) func(message string, err error) {
	return func(message string, err error) {
		if err == nil {
			uclog.Infof(ctx, "[Statsig] %s", message)
		} else if logEventFailureExpression.MatchString(message) {
			// We don't want to log this error, but we don't care about it too much. we have a metric we can alert on if it happens too much
			failedEventUpload.WithLabelValues().Inc()
			uclog.Warningf(ctx, "[Statsig] %s - %v", message, err)
		} else {
			uclog.Errorf(ctx, "[Statsig] %s - %v", message, err)
		}
	}
}

func geRulesUpdatedCallback(ctx context.Context) func(rules string, time int64) {
	return func(rules string, time int64) {
		// Not sure what to do with this yet, but we can log it for now.
		uclog.Infof(ctx, "[Statsig] rules update: %s time: %v", rules, time)
	}
}

// Close shuts down the feature flags package and allows syncing and flushing data to statsig
func Close(ctx context.Context) {
	if !isInitialized(ctx) {
		return
	}
	uclog.Infof(ctx, "Shutting down Statsig")
	statsig.Shutdown()
}

// Init initializes the feature flags package w/ a given API key for a vendor (statsig)
func Init(ctx context.Context, cfg *Config) {
	if cfg == nil {
		notInitWarningLogged = true
		uclog.Warningf(ctx, "No feature flag config - all features flags will be disabled")
		return
	}
	unv := universe.Current()
	apiKey, err := cfg.APIKey.Resolve(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Failed to resolve statsig API key secret : %v", err)
		return
	}
	options := &statsig.Options{
		LocalMode:            unv.IsTestOrCI(),
		RulesUpdatedCallback: geRulesUpdatedCallback(ctx),
		OutputLoggerOptions: statsig.OutputLoggerOptions{
			LogCallback:            getLogCallback(ctx),
			DisableInitDiagnostics: !cfg.VerboseLogging,
			DisableSyncDiagnostics: !cfg.VerboseLogging,
		},
		Environment: statsig.Environment{
			Tier: string(unv),
		},
	}
	uclog.Infof(ctx, "Init Statsig with APIKey (hash): '%s' options: %+v", crypto.GetMD5Hash(apiKey), options)
	initLock.Lock()
	defer initLock.Unlock()
	statsig.InitializeWithOptions(apiKey, options)
}

func isInitialized(ctx context.Context) bool {
	// statsig will panic if the CheckGate API is called and ths SDK is not initialized.
	// We don't want that behavior in our app, so instead we will short-circuit, return false (feature flag off) and log a warning
	initLock.RLock()
	defer initLock.RUnlock()
	res := statsig.IsInitialized()
	if !res && !notInitWarningLogged {
		notInitWarningLogged = true
		if universe.Current().IsCloud() {
			uclog.Errorf(ctx, "Statsig is not initialized, feature flags will default to false")
		} else {
			uclog.Warningf(ctx, "Statsig is not initialized, feature flags will default to false")
		}
	}
	return res
}

// IsEnabledGlobally returns whether a feature flag is enabled globally (per environment) regardless of tenant/context.
func IsEnabledGlobally(ctx context.Context, flag Flag) bool {
	if !isInitialized(ctx) {
		return false
	}
	reqID := request.GetRequestID(ctx)
	ssUser := getUser(scopeGlobal, reqID.String())
	return statsig.CheckGate(ssUser, string(flag))
}

// IsEnabledForTenant returns whether a feature flag is enabled for a given tenant ID
func IsEnabledForTenant(ctx context.Context, flag Flag, tenantID uuid.UUID) bool {
	if tenantID.IsNil() || !isInitialized(ctx) {
		return false
	}
	ssUser := getUserForID(scopeTenant, tenantID)
	return statsig.CheckGate(ssUser, string(flag))
}

// IsEnabledForCompany returns true if the feature flag is enabled for the given company ID
func IsEnabledForCompany(ctx context.Context, flag Flag, companyID uuid.UUID) bool {
	if companyID.IsNil() || !isInitialized(ctx) {
		return false
	}
	ssUser := getUserForID(scopeCompany, companyID)
	return statsig.CheckGate(ssUser, string(flag))
}

// IsEnabledForHostname returns true if the feature flag is enabled for the given hostname
func IsEnabledForHostname(ctx context.Context, flag Flag, hostname string) bool {
	if hostname == "" || !isInitialized(ctx) {
		return false
	}
	ssUser := getUser(scopeHostname, hostname)
	v := statsig.CheckGate(ssUser, string(flag))
	return v
}

func getUserForID(scope flagScope, id uuid.UUID) statsig.User {
	return getUser(scope, id.String())
}

func getUser(scope flagScope, id string) statsig.User {
	idKey := fmt.Sprintf("%vID", scope)
	return statsig.User{
		// Statsig only allows to scope feature flags to users, so we need to make a "fake" user for different scope types
		UserID: fmt.Sprintf("%v_%v_%v", userIDSchemeVersion, scope, id),
		Custom: map[string]any{
			"scope": string(scope),
			idKey:   id,
		},
	}
}

// GetStringsListForTenant returns a list of strings for a given tenant ID and a dynamic config
func GetStringsListForTenant(ctx context.Context, dcName DynamicConfig, tenantID uuid.UUID) []string {
	// https://docs.statsig.com/server/golangSDK/#reading-a-dynamic-config
	if !isInitialized(ctx) {
		return nil
	}
	ssUser := getUserForID(scopeTenant, tenantID)
	dc := statsig.GetConfig(ssUser, string(dcName))
	stringList := dc.GetSlice("value", nil)
	// We just want to log errors, we don't want to propagate them
	if stringList == nil {
		uclog.Errorf(ctx, "Failed to get string list for dynamic config %s", dcName)
		return nil
	}
	strings := make([]string, 0, len(stringList))
	for _, s := range stringList {
		currStr, ok := s.(string)
		if !ok {
			uclog.Errorf(ctx, "Failed to convert %T (%v) to string", s, s)
			return nil
		}
		strings = append(strings, currStr)
	}
	return strings
}
