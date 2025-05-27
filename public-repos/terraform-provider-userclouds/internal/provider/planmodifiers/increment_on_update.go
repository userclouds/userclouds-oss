package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IncrementOnUpdate increments the value of the field whenever the resource is updated
func IncrementOnUpdate() planmodifier.Int64 {
	return incrementOnUpdateModifier{}
}

type incrementOnUpdateModifier struct{}

const description = "Increments value whenever resource is updated"

func (m incrementOnUpdateModifier) Description(_ context.Context) string {
	return description
}

func (m incrementOnUpdateModifier) MarkdownDescription(_ context.Context) string {
	return description
}

func (m incrementOnUpdateModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Do not increment on resource creation
	if req.State.Raw.IsNull() {
		return
	}

	// Do not increment on resource destroy
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do not increment if there was no resource change
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	resp.PlanValue = types.Int64Value(req.StateValue.ValueInt64() + 1)
}
