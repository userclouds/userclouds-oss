package paths

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// Path constants for the userstore
var (
	SendInvitePath      = "/invite/send"
	CreateUserPath      = "/create/submit"
	LoginPath           = "/login"
	ImpersonateUserPath = "/impersonateuser"

	BaseLoginAppPath = "/loginapp/register"
	GetLoginAppPath  = func(id uuid.UUID) string {
		return fmt.Sprintf("%s/%s", BaseLoginAppPath, id)
	}
	CreateLoginAppPath = BaseLoginAppPath
	DeleteLoginAppPath = GetLoginAppPath
	UpdateLoginAppPath = GetLoginAppPath
	ListLoginAppPath   = func(organizationID uuid.UUID) string {
		url := BaseLoginAppPath
		if organizationID != uuid.Nil {
			url += "?organization_id=" + organizationID.String()
		}
		return url
	}
)
