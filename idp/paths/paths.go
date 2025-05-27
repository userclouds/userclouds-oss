package paths

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

// Path constants for the userstore
var (
	IDPBasePath = "/authn" // TODO change this

	// TODO: finish converting IDP path handling to use these
	CreateUser     = fmt.Sprintf("%s/users", IDPBasePath)
	AddAuthnToUser = fmt.Sprintf("%s/addauthntouser", IDPBasePath)

	UserStoreBasePath = "/userstore"

	BaseConfigPath = fmt.Sprintf("%s/config", UserStoreBasePath)

	ListUserRegionsPath = fmt.Sprintf("%s/regions", BaseConfigPath)

	DatabasePath       = fmt.Sprintf("%s/databases", BaseConfigPath)
	CreateDatabasePath = DatabasePath
	ListDatabasesPath  = DatabasePath
	singleDatabasePath = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", DatabasePath, id) }
	GetDatabasePath    = singleDatabasePath
	UpdateDatabasePath = singleDatabasePath
	DeleteDatabasePath = singleDatabasePath
	TestDatabasePath   = fmt.Sprintf("%s/test", DatabasePath)

	ObjectStorePath       = fmt.Sprintf("%s/objectstores", BaseConfigPath)
	CreateObjectStorePath = ObjectStorePath
	ListObjectStoresPath  = ObjectStorePath
	singleObjectStorePath = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", ObjectStorePath, id) }
	GetObjectStorePath    = singleObjectStorePath
	UpdateObjectStorePath = singleObjectStorePath
	DeleteObjectStorePath = singleObjectStorePath

	BaseConfigColumnsPath  = fmt.Sprintf("%s/columns", BaseConfigPath)
	singleConfigColumnPath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseConfigColumnsPath, id)
	}
	CreateColumnPath = BaseConfigColumnsPath
	DeleteColumnPath = singleConfigColumnPath
	GetColumnPath    = singleConfigColumnPath
	ListColumnsPath  = BaseConfigColumnsPath
	UpdateColumnPath = singleConfigColumnPath

	BaseConfigAccessorPath   = fmt.Sprintf("%s/accessors", BaseConfigPath)
	singleConfigAccessorPath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseConfigAccessorPath, id)
	}
	versionedSingleConfigAccessorPath = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?accessor_version=%d", BaseConfigAccessorPath, id, version)
	}
	CreateAccessorPath       = BaseConfigAccessorPath
	DeleteAccessorPath       = singleConfigAccessorPath
	GetAccessorPath          = singleConfigAccessorPath
	GetAccessorByVersionPath = versionedSingleConfigAccessorPath
	ListAccessorsPath        = BaseConfigAccessorPath
	UpdateAccessorPath       = singleConfigAccessorPath

	BaseAccessorPath    = fmt.Sprintf("%s/accessors", BaseAPIPath)
	ExecuteAccessorPath = BaseAccessorPath

	BaseConfigMutatorPath   = fmt.Sprintf("%s/mutators", BaseConfigPath)
	singleConfigMutatorPath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseConfigMutatorPath, id)
	}
	versionedSingleConfigMutatorPath = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?mutator_version=%d", BaseConfigMutatorPath, id, version)
	}
	CreateMutatorPath       = BaseConfigMutatorPath
	DeleteMutatorPath       = singleConfigMutatorPath
	GetMutatorPath          = singleConfigMutatorPath
	GetMutatorByVersionPath = versionedSingleConfigMutatorPath
	ListMutatorsPath        = BaseConfigMutatorPath
	UpdateMutatorPath       = singleConfigMutatorPath

	BaseMutatorPath    = fmt.Sprintf("%s/mutators", BaseAPIPath)
	ExecuteMutatorPath = BaseMutatorPath

	BaseConfigPurposePath   = fmt.Sprintf("%s/purposes", BaseConfigPath)
	singleConfigPurposePath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseConfigPurposePath, id)
	}

	CreatePurposePath = BaseConfigPurposePath
	ListPurposesPath  = BaseConfigPurposePath
	GetPurposePath    = singleConfigPurposePath
	DeletePurposePath = singleConfigPurposePath
	UpdatePurposePath = singleConfigPurposePath

	CreateUserWithMutatorPath       = fmt.Sprintf("%s/users", BaseAPIPath)
	GetConsentedPurposesForUserPath = fmt.Sprintf("%s/consentedpurposes", BaseAPIPath)

	BaseAPIPath = fmt.Sprintf("%s/api", UserStoreBasePath)

	TokenizerBasePath = "/tokenizer"

	BaseTokenPath        = fmt.Sprintf("%s/tokens", TokenizerBasePath)
	CreateToken          = BaseTokenPath
	DeleteToken          = BaseTokenPath
	ResolveToken         = fmt.Sprintf("%s/actions/resolve", BaseTokenPath)
	InspectToken         = fmt.Sprintf("%s/actions/inspect", BaseTokenPath)
	LookupToken          = fmt.Sprintf("%s/actions/lookup", BaseTokenPath)
	LookupOrCreateTokens = fmt.Sprintf("%s/actions/lookuporcreate", BaseTokenPath)

	BasePolicyPath = fmt.Sprintf("%s/policies", TokenizerBasePath)

	BaseAccessPolicyPath  = fmt.Sprintf("%s/access", BasePolicyPath)
	ListAccessPolicies    = BaseAccessPolicyPath
	GetAccessPolicyByName = func(name string) string {
		return fmt.Sprintf("%s?policy_name=%s", BaseAccessPolicyPath, name)
	}
	GetAccessPolicyByNameAndVersion = func(name string, version int) string {
		return fmt.Sprintf("%s?policy_name=%s&policy_version=%d", BaseAccessPolicyPath, name, version)
	}
	GetAccessPolicy = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseAccessPolicyPath, id)
	}
	GetAccessPolicyByVersion = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?policy_version=%d", BaseAccessPolicyPath, id, version)
	}
	CreateAccessPolicy = BaseAccessPolicyPath
	UpdateAccessPolicy = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", BaseAccessPolicyPath, id) }
	DeleteAccessPolicy = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?policy_version=%d", BaseAccessPolicyPath, id, version)
	}
	DeleteAllAccessPolicyVersions = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s?policy_version=all", BaseAccessPolicyPath, id)
	}
	TestAccessPolicy    = fmt.Sprintf("%s/actions/test", BaseAccessPolicyPath)
	ExecuteAccessPolicy = fmt.Sprintf("%s/actions/execute", BaseAccessPolicyPath)

	BaseAccessPolicyTemplatePath  = fmt.Sprintf("%s/accesstemplate", BasePolicyPath)
	ListAccessPolicyTemplates     = BaseAccessPolicyTemplatePath
	GetAccessPolicyTemplateByName = func(name string) string {
		return fmt.Sprintf("%s?template_name=%s", BaseAccessPolicyTemplatePath, name)
	}
	GetAccessPolicyTemplateByNameAndVersion = func(name string, version int) string {
		return fmt.Sprintf("%s?template_name=%s&template_version=%d", BaseAccessPolicyTemplatePath, name, version)
	}
	GetAccessPolicyTemplate = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseAccessPolicyTemplatePath, id)
	}
	GetAccessPolicyTemplateByVersion = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?template_version=%d", BaseAccessPolicyTemplatePath, id, version)
	}
	CreateAccessPolicyTemplate = BaseAccessPolicyTemplatePath
	UpdateAccessPolicyTemplate = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", BaseAccessPolicyTemplatePath, id) }
	DeleteAccessPolicyTemplate = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?template_version=%d", BaseAccessPolicyTemplatePath, id, version)
	}
	DeleteAllAccessPolicyTemplateVersions = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s?template_version=all", BaseAccessPolicyTemplatePath, id)
	}
	TestAccessPolicyTemplate = fmt.Sprintf("%s/actions/test", BaseAccessPolicyTemplatePath)

	BaseTransformerPath  = fmt.Sprintf("%s/transformation", BasePolicyPath)
	ListTransformers     = BaseTransformerPath
	GetTransformerByName = func(name string) string {
		return fmt.Sprintf("%s?transformer_name=%s", BaseTransformerPath, name)
	}
	GetTransformer = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseTransformerPath, id)
	}
	GetTransformerByVersion = func(id uuid.UUID, version int) string {
		return fmt.Sprintf("%s/%s?transformer_version=%d", BaseTransformerPath, id, version)
	}
	GetTransformerByNameAndVersion = func(name string, version int) string {
		return fmt.Sprintf("%s?transformer_name=%s&transformer_version=%d", BaseTransformerPath, name, version)
	}
	CreateTransformer  = BaseTransformerPath
	UpdateTransformer  = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", BaseTransformerPath, id) }
	DeleteTransformer  = func(id uuid.UUID) string { return fmt.Sprintf("%s/%s", BaseTransformerPath, id) }
	TestTransformer    = fmt.Sprintf("%s/actions/test", BaseTransformerPath)
	ExecuteTransformer = fmt.Sprintf("%s/actions/execute", BaseTransformerPath)

	BaseDataMappingPath = "/userstore/datamapping"

	BaseSecretPath = fmt.Sprintf("%s/secret", BasePolicyPath)
	ListSecrets    = BaseSecretPath
	CreateSecret   = BaseSecretPath
	DeleteSecret   = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseSecretPath, id)
	}

	CreateDataSourcePath = fmt.Sprintf("%s/datasource", BaseDataMappingPath)
	singleDataSourcePath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", CreateDataSourcePath, id)
	}
	DeleteDataSourcePath = singleDataSourcePath
	GetDataSourcePath    = singleDataSourcePath
	UpdateDataSourcePath = singleDataSourcePath
	ListDataSourcesPath  = CreateDataSourcePath

	CreateDataSourceElementPath = fmt.Sprintf("%s/element", BaseDataMappingPath)
	singleDataSourceElementPath = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", CreateDataSourceElementPath, id)
	}
	DeleteDataSourceElementPath = singleDataSourceElementPath
	GetDataSourceElementPath    = singleDataSourceElementPath
	UpdateDataSourceElementPath = singleDataSourceElementPath
	ListDataSourceElementsPath  = CreateDataSourceElementPath

	DownloadGolangSDKPath     = fmt.Sprintf("%s/download/codegensdk.go", UserStoreBasePath)
	DownloadPythonSDKPath     = fmt.Sprintf("%s/download/codegensdk.py", UserStoreBasePath)
	DownloadTypescriptSDKPath = fmt.Sprintf("%s/download/codegensdk.ts", UserStoreBasePath)

	ExternalOIDCIssuersPath = fmt.Sprintf("%s/oidcissuers", UserStoreBasePath)
)

// StripUserstoreBase makes the URLs functional for handler setup
func StripUserstoreBase(path string) string {
	return strings.TrimPrefix(path, UserStoreBasePath)
}

// StripTokenizerBase makes the URLs functional for handler setup
func StripTokenizerBase(path string) string {
	return strings.TrimPrefix(path, TokenizerBasePath)
}

// GetReferenceURLForAccessor return URL pointing at a particular access policy object
func GetReferenceURLForAccessor(id uuid.UUID, v int) string {
	return fmt.Sprintf("%s/%s/%d", BaseAccessorPath, id.String(), v)
}

// GetReferenceURLForMutator return URL pointing at a particular transformer object
func GetReferenceURLForMutator(id uuid.UUID, v int) string {
	return fmt.Sprintf("%s/%s/%d", BaseMutatorPath, id.String(), v)
}

// GetReferenceURLForAccessPolicy return URL pointing at a particular access policy object
func GetReferenceURLForAccessPolicy(id uuid.UUID, v int) string {
	return fmt.Sprintf("%s/%s/%d", BaseAccessPolicyPath, id.String(), v)
}

// GetReferenceURLForTransformer return URL pointing at a particular transformer object
func GetReferenceURLForTransformer(id uuid.UUID, v int) string {
	return fmt.Sprintf("%s/%s/%d", BaseTransformerPath, id.String(), v)
}
