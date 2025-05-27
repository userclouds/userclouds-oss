package internal

import "userclouds.com/internal/auditlog"

// auditlog.EntryType constants for IDP
const (
	// Userstore management API audit log types
	AuditLogEventTypeCreateDataType auditlog.EventType = "CreateDataType"
	AuditLogEventTypeUpdateDataType auditlog.EventType = "UpdateDataType"
	AuditLogEventTypeDeleteDataType auditlog.EventType = "DeleteDataType"

	AuditLogEventTypeCreateColumnConfig auditlog.EventType = "CreateColumnConfig"
	AuditLogEventTypeUpdateColumnConfig auditlog.EventType = "UpdateColumnConfig"
	AuditLogEventTypeDeleteColumnConfig auditlog.EventType = "DeleteColumnConfig"

	AuditLogEventTypeCreateAccessorConfig auditlog.EventType = "CreateAccessorConfig"
	AuditLogEventTypeUpdateAccessorConfig auditlog.EventType = "UpdateAccessorConfig"
	AuditLogEventTypeDeleteAccessorConfig auditlog.EventType = "DeleteAccessorConfig"

	AuditLogEventTypeCreateMutatorConfig auditlog.EventType = "CreateMutatorConfig"
	AuditLogEventTypeUpdateMutatorConfig auditlog.EventType = "UpdateMutatorConfig"
	AuditLogEventTypeDeleteMutatorConfig auditlog.EventType = "DeleteMutatorConfig"

	AuditLogEventTypeCreateToken  auditlog.EventType = "CreateToken"
	AuditLogEventTypeResolveToken auditlog.EventType = "ResolveToken"
	AuditLogEventTypeDeleteToken  auditlog.EventType = "DeleteToken"
	AuditLogEventTypeInspectToken auditlog.EventType = "InspectToken"
	AuditLogEventTypeLookupToken  auditlog.EventType = "LookupToken"

	AuditLogEventTypeCreateAccessPolicy auditlog.EventType = "CreateAccessPolicy"
	AuditLogEventTypeUpdateAccessPolicy auditlog.EventType = "UpdateAccessPolicy"
	AuditLogEventTypeDeleteAccessPolicy auditlog.EventType = "DeleteAccessPolicy"

	AuditLogEventTypeCreateTransformer auditlog.EventType = "CreateTransformer"
	AuditLogEventTypeUpdateTransformer auditlog.EventType = "UpdateTransformer"
	AuditLogEventTypeDeleteTransformer auditlog.EventType = "DeleteTransformer"

	AuditLogEventTypeCreateUserSearchIndex         auditlog.EventType = "CreateUserSearchIndex"
	AuditLogEventTypeDeleteUserSearchIndex         auditlog.EventType = "DeleteUserSearchIndex"
	AuditLogEventTypeUpdateUserSearchIndex         auditlog.EventType = "UpdateUserSearchIndex"
	AuditLogEventTypeRemoveAccessorUserSearchIndex auditlog.EventType = "RemoveAccessorUserSearchIndex"
	AuditLogEventTypeSetAccessorUserSearchIndex    auditlog.EventType = "SetAccessorUserSearchIndex"

	AuditLogEventTypeExecuteAccessor       auditlog.EventType = "ExecuteAccessor"
	AuditLogEventTypeSqlshimUnhandledQuery auditlog.EventType = "UnhandledQuery"
)
