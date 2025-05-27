package internal

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
)

// CheckAttributeViaService checks if a source object has a specific attribute on a target object
func CheckAttributeViaService(ctx context.Context, checkAttributeServiceName string, tenantID, sourceObjectID, targetObjectID uuid.UUID, attributeName string) (bool, []authz.AttributePathNode, error) {
	ctx = request.NewRequestID(ctx)

	var host string
	if universe.Current().IsDev() {
		host = "http://localhost:5210"
	} else {
		host = fmt.Sprintf("http://%s", kubernetes.GetHostFromServiceName(checkAttributeServiceName))
	}
	client := jsonclient.New(host)

	var resp authz.CheckAttributeResponse
	query := url.Values{}
	query.Add("source_object_id", sourceObjectID.String())
	query.Add("target_object_id", targetObjectID.String())
	query.Add("attribute", attributeName)
	if err := client.Get(ctx, fmt.Sprintf("/checkattribute/%s?%s", tenantID, query.Encode()), &resp); err != nil {
		return false, nil, ucerr.Wrap(err)
	}

	return resp.HasAttribute, resp.Path, nil
}
