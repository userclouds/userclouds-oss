package paths

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// SearchIndexPath is the path constant for interacting with user search indices
var SearchIndexPath = fmt.Sprintf("%s/%s", BaseConfigPath, "searchindices")

// RemoveAccessorSearchIndexPath is the path constant for removing an accessor user search
// index association
var RemoveAccessorSearchIndexPath = fmt.Sprintf("%s/%s", SearchIndexPath, "accessor/remove")

// SetAccessorSearchIndexPath is the path constant for setting an accessor user search
// index association
var SetAccessorSearchIndexPath = fmt.Sprintf("%s/%s", SearchIndexPath, "accessor/set")

// SearchIndexPathForID returns the user search index path for a specific user search index id
func SearchIndexPathForID(indexID uuid.UUID) string {
	return fmt.Sprintf("%s/%v", SearchIndexPath, indexID)
}
