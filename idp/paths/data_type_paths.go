package paths

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// DataTypePath is the path constant for interacting with data types
var DataTypePath = fmt.Sprintf("%s/%s", BaseConfigPath, "datatypes")

// DataTypePathForID returns the data type path for a specific data type id
func DataTypePathForID(dataTypeID uuid.UUID) string {
	return fmt.Sprintf("%s/%v", DataTypePath, dataTypeID)
}
