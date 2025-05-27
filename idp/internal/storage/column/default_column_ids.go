package column

import "github.com/gofrs/uuid"

// system columns

// IDColumnID is the column id of the system ID column
var IDColumnID = uuid.Must(uuid.FromString("b1d12a3e-dbf7-4405-b5fc-7e3919d8e089"))

// CreatedColumnID is the column id of the system Created column
var CreatedColumnID = uuid.Must(uuid.FromString("23843bce-b9f1-4d32-9dad-a8c15bc9a17e"))

// UpdatedColumnID is the column id of the system Updated column
var UpdatedColumnID = uuid.Must(uuid.FromString("58616e49-ab3f-48ad-9209-d4b232a35a6d"))

// OrganizationColumnID is the column id of the system Organization column
var OrganizationColumnID = uuid.Must(uuid.FromString("15772f46-12c1-49af-9e01-4d38545a70bd"))

// VersionColumnID is the column id of the system Version column
var VersionColumnID = uuid.Must(uuid.FromString("f94923b4-e403-4c95-bcbe-481aaba5fa63"))

// default columns

// NameColumnID is the column id of the default Name column
var NameColumnID = uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794"))

// NicknameColumnID is the column id of the default Nickname column
var NicknameColumnID = uuid.Must(uuid.FromString("83cc42b0-da8c-4a61-9db1-da70f21bab60"))

// PictureColumnID is the ID of the default Picture column
var PictureColumnID = uuid.Must(uuid.FromString("4d4d0757-3bc2-424d-9caf-a930edb49b69"))

// EmailColumnID is the ID of the default Email column
var EmailColumnID = uuid.Must(uuid.FromString("2c7a7c9b-90e8-47e4-8f6e-ec73bd2dec16"))

// EmailVerifiedColumnID is the ID of the default EmailVerified column
var EmailVerifiedColumnID = uuid.Must(uuid.FromString("12b3f133-4ad1-4f11-9d7d-313eb7cb95fa"))

// ExternalAliasColumnID is the ID of the default ExternalAlias column
var ExternalAliasColumnID = uuid.Must(uuid.FromString("2ee3d57d-9756-464e-a5e9-04244936cb9e"))
