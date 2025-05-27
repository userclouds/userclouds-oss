package authz

import (
	"fmt"

	"userclouds.com/infra/pagination"
)

func (ot ObjectType) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "type_name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"type_name:%v,id:%v",
				ot.TypeName,
				ot.ID,
			),
		)
	}
}

func (ObjectType) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"type_name": pagination.StringKeyType,
		"created":   pagination.TimestampKeyType,
		"updated":   pagination.TimestampKeyType,
	}
}

//go:generate genpageable ObjectType

func (et EdgeType) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "type_name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"type_name:%v,id:%v",
				et.TypeName,
				et.ID,
			),
		)
	}
}

func (EdgeType) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"type_name":             pagination.StringKeyType,
		"organization_id":       pagination.UUIDKeyType,
		"source_object_type_id": pagination.UUIDKeyType,
		"target_object_type_id": pagination.UUIDKeyType,
		"created":               pagination.TimestampKeyType,
		"updated":               pagination.TimestampKeyType,
	}
}

//go:generate genpageable EdgeType

func (Object) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"alias":           pagination.StringKeyType,
		"organization_id": pagination.UUIDKeyType,
		"type_id":         pagination.UUIDKeyType,
		"created":         pagination.TimestampKeyType,
		"updated":         pagination.TimestampKeyType,
	}
}

//go:generate genpageable Object

func (Edge) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"source_object_id": pagination.UUIDKeyType,
		"target_object_id": pagination.UUIDKeyType,
		"created":          pagination.TimestampKeyType,
		"updated":          pagination.TimestampKeyType,
	}
}

//go:generate genpageable Edge

func (Organization) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

//go:generate genpageable Organization
