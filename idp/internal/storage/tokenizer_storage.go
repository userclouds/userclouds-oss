package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
)

// ErrStillInUse represents an error for an object being deleted that is still in use
var ErrStillInUse = ucerr.New("entity still in use")

// GetLatestAccessPolicyTemplatesPaginated returns the latest version of all access policies, paginated
func (s Storage) GetLatestAccessPolicyTemplatesPaginated(ctx context.Context, p pagination.Paginator) ([]AccessPolicyTemplate, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf(`SELECT created, deleted, description, function, id, name, updated, version, is_system FROM (
		SELECT created, deleted, description, function, id, name, updated, version, is_system FROM (
			SELECT a.created, a.deleted, a.description, a.function, a.id, a.name, a.updated, a.version, a.is_system FROM access_policy_templates AS a
			JOIN
			(select id, max(version) version FROM access_policy_templates WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
			ON a.id = b.id AND a.version=b.version) tmp
		WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp2
	ORDER BY %s; /* lint-sql-unsafe-columns */`, p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())
	var objs []AccessPolicyTemplate
	if err := s.db.SelectContext(ctx, "GetLatestAccessPolicyTemplatesPaginated", &objs, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	objs, respFields := pagination.ProcessResults(objs, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// GetAllAccessPolicyTemplateVersions returns all versions of a given access policy
func (s Storage) GetAllAccessPolicyTemplateVersions(ctx context.Context, id uuid.UUID) ([]AccessPolicyTemplate, error) {
	const q = `SELECT created, deleted, description, function, id, name, updated, version, is_system FROM access_policy_templates WHERE id=$1 AND deleted='0001-01-01 00:00:00';`

	var apts []AccessPolicyTemplate
	if err := s.db.SelectContext(ctx, "GetAllAccessPolicyTemplateVersions", &apts, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return apts, nil
}

// GetAccessPolicyTemplateByVersion looks up a specific version of an access policy
func (s Storage) GetAccessPolicyTemplateByVersion(ctx context.Context, id uuid.UUID, version int) (*AccessPolicyTemplate, error) {
	const q = `SELECT created, deleted, description, function, id, name, updated, version, is_system FROM access_policy_templates WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00';`

	var apt AccessPolicyTemplate
	if err := s.db.GetContext(ctx, "GetAccessPolicyTemplateByVersion", &apt, q, id, version); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &apt, nil
}

// GetAccessPolicyTemplateByName is used for de-duping to ensure names are not reused, it returns the latest version of the access policy with the given name
func (s Storage) GetAccessPolicyTemplateByName(ctx context.Context, name string) (*AccessPolicyTemplate, error) {
	const q = `
/* lint-sql-allow-multi-column-aggregation */
SELECT a.created, a.deleted, a.description, a.function, a.id, a.name, a.updated, a.version, a.is_system
FROM access_policy_templates AS a
JOIN (SELECT id, name, MAX(version) version FROM access_policy_templates WHERE deleted='0001-01-01 00:00:00' GROUP BY id, name) AS b
ON a.id=b.id
AND a.name=b.name
AND a.version=b.version
WHERE a.deleted='0001-01-01 00:00:00'
AND LOWER(a.name)=LOWER($1);`

	var apt AccessPolicyTemplate
	if err := s.db.GetContext(ctx, "GetAccessPolicyTemplateByName", &apt, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &apt, nil
}

// checks that the access policy template version is unused, pass in -1 for version to check all versions
func (s Storage) preDeleteAccessPolicyTemplate(ctx context.Context, id uuid.UUID, version int) error {
	if version != -1 {
		const q1 = `SELECT version FROM access_policy_templates WHERE id=$1 AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns */`

		var versions []int
		if err := s.db.SelectContext(ctx, "checkAccessPolicyTemplateUnused.templates", &versions, q1, id); err != nil {
			return ucerr.Wrap(err)
		}

		// if we're not deleting the last version, we don't need to check for usage
		if len(versions) != 1 || versions[0] != version {
			return nil
		}
	}

	const q2 = `SELECT id FROM access_policies WHERE component_ids @> $1::uuid[] AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns */`

	var ids []uuid.UUID
	if err := s.db.SelectContext(ctx, "checkAccessPolicyTemplateUnused.policies", &ids, q2, uuidarray.UUIDArray{id}); err != nil {
		return ucerr.Wrap(err)
	}

	if len(ids) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy template is in use by %d access policies", len(ids))
	}

	return nil
}

// GetLatestAccessPoliciesPaginated returns the latest version of all access policies, paginated
func (s Storage) GetLatestAccessPoliciesPaginated(ctx context.Context, p pagination.Paginator) ([]AccessPolicy, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf(`SELECT id, updated, deleted, name, description, policy_type, tag_ids, version, component_ids, component_parameters, component_types, created, is_system, is_autogenerated, metadata FROM (
		SELECT id, updated, deleted, name, description, policy_type, tag_ids, version, component_ids, component_parameters, component_types, created, is_system, is_autogenerated, metadata FROM (
			SELECT a.component_ids, a.component_parameters, a.component_types, a.created, a.deleted, a.description, a.id, a.name, a.policy_type, a.tag_ids, a.updated, a.version, a.is_system, a.is_autogenerated, a.metadata FROM access_policies AS a
			JOIN
			(select id, max(version) version FROM access_policies WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
			ON a.id = b.id AND a.version=b.version) tmp
		WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp2
	ORDER BY %s; /* lint-sql-unsafe-columns */`, p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objs []AccessPolicy
	if err := s.db.SelectContext(ctx, "GetLatestAccessPoliciesPaginated", &objs, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	objs, respFields := pagination.ProcessResults(objs, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// GetAllAccessPolicyVersions returns all versions of a given access policy
func (s Storage) GetAllAccessPolicyVersions(ctx context.Context, id uuid.UUID) ([]AccessPolicy, error) {
	const q = `SELECT component_ids, component_parameters, component_types, created, deleted, description, id, name, policy_type, tag_ids, updated, version, is_system, is_autogenerated, metadata, thresholds FROM access_policies WHERE id=$1 AND deleted='0001-01-01 00:00:00';`

	var aps []AccessPolicy
	if err := s.db.SelectContext(ctx, "GetAllAccessPolicyVersions", &aps, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return aps, nil
}

// GetAccessPolicyByVersion looks up a specific version of an access policy
func (s Storage) GetAccessPolicyByVersion(ctx context.Context, id uuid.UUID, version int) (*AccessPolicy, error) {
	const q = `SELECT component_ids, component_parameters, component_types, created, deleted, description, id, name, policy_type, tag_ids, updated, version, is_system, is_autogenerated, metadata, thresholds FROM access_policies WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00';`

	var ap AccessPolicy
	if err := s.db.GetContext(ctx, "GetAccessPolicyByVersion", &ap, q, id, version); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &ap, nil
}

// GetAccessPolicyByName is used for de-duping to ensure names are not reused, it returns the latest version of the access policy with the given name
// TODO needs to be redone once we add hierarchy
func (s Storage) GetAccessPolicyByName(ctx context.Context, name string) (*AccessPolicy, error) {
	const q = `
/* lint-sql-allow-multi-column-aggregation */
SELECT a.component_ids, a.component_parameters, a.component_types, a.created, a.deleted, a.description, a.id, a.name, a.policy_type, a.tag_ids, a.updated, a.version, a.is_system, a.is_autogenerated, a.metadata, a.thresholds
FROM access_policies AS a
JOIN (SELECT id, name, MAX(version) version FROM access_policies WHERE deleted='0001-01-01 00:00:00' GROUP BY id, name) AS b
ON a.id=b.id
AND a.name=b.name
AND a.version=b.version
WHERE a.deleted='0001-01-01 00:00:00'
AND LOWER(a.name)=LOWER($1);`

	var ap AccessPolicy
	if err := s.db.GetContext(ctx, "GetAccessPolicyByName", &ap, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &ap, nil
}

// GetAccessPoliciesForResourceIDs returns a list of access policies for the given resource ids
func (s Storage) GetAccessPoliciesForResourceIDs(ctx context.Context, errorOnMissing bool, accessPolicyRIDs ...userstore.ResourceID) ([]AccessPolicy, error) {
	var ids []uuid.UUID
	var names []string

	for _, rid := range accessPolicyRIDs {
		if rid.ID != uuid.Nil {
			ids = append(ids, rid.ID)
		} else if rid.Name != "" {
			names = append(names, strings.ToLower(rid.Name))
		} else {
			return nil, ucerr.Friendlyf(nil, "invalid access policy resource id: %+v", rid)
		}
	}

	const q = `
SELECT component_ids, component_parameters, component_types, created, deleted, description, id, name, policy_type, tag_ids, updated, version, is_system, is_autogenerated, metadata, thresholds
FROM access_policies
WHERE (id=ANY($1) OR LOWER(name)=ANY($2))
AND deleted='0001-01-01 00:00:00';`

	var objects []AccessPolicy
	if err := s.db.SelectContext(ctx, "GetAccessPoliciesForResourceIDs", &objects, q, pq.Array(ids), pq.Array(names)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	found := make([]bool, len(accessPolicyRIDs))
	for _, obj := range objects {
		for i, rid := range accessPolicyRIDs {
			if obj.ID == rid.ID || strings.EqualFold(obj.Name, rid.Name) {
				found[i] = true
				if rid.Name != "" && !strings.EqualFold(obj.Name, rid.Name) {
					return nil, ucerr.Errorf("access policy name mismatch for resource ID: %v, got %s", rid, obj.Name)
				}
				if rid.ID != uuid.Nil && obj.ID != rid.ID {
					return nil, ucerr.Errorf("access policy ID mismatch for resource ID: %v, got %s", rid, obj.ID)
				}
			}
		}
	}

	if errorOnMissing {
		missingRIDs := []string{}
		for i := range found {
			if !found[i] {
				missingRIDs = append(missingRIDs, fmt.Sprintf("%+v", accessPolicyRIDs[i]))
			}
		}

		if len(missingRIDs) > 0 {
			return nil, ucerr.Errorf("Not all requested IDs where loaded. Missing: [%s]", strings.Join(missingRIDs, ", "))
		}
	}
	return objects, nil
}

// CheckAccessPolicyUnused checks if an access policy is in use by any accessors, mutators, or token_records
func (s Storage) CheckAccessPolicyUnused(ctx context.Context, id uuid.UUID) error {
	const q1 = `SELECT name FROM (
		SELECT a.name, a.access_policy_id, a.token_access_policy_ids FROM accessors AS a
		JOIN (select id, max(version) version
		FROM accessors
		WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
		ON a.id = b.id AND a.version=b.version) AS c
	WHERE (access_policy_id=$1 OR $1::UUID=ANY(token_access_policy_ids)); /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	var names []string
	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.accessors", &names, q1, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(names) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by the following %d accessors: %s", len(names), strings.Join(names, ", "))
	}

	const q2 = `SELECT name FROM (
		SELECT a.name, a.access_policy_id FROM mutators AS a
		JOIN (select id, max(version) version
		FROM mutators
		WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
		ON a.id = b.id AND a.version=b.version) AS c
	WHERE access_policy_id=$1; /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.mutators", &names, q2, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(names) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by following %d mutators: %s", len(names), strings.Join(names, ", "))
	}

	const q3 = `SELECT id FROM token_records WHERE access_policy_id=$1 AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns */`

	var ids []uuid.UUID
	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.tokens", &ids, q3, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(ids) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by %d tokens", len(ids))
	}

	const q4 = `SELECT name FROM (
		SELECT a.name, a.component_ids FROM access_policies AS a
		JOIN (select id, max(version) version
		FROM access_policies
		WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
		ON a.id = b.id AND a.version=b.version) AS c
	WHERE $1::UUID=ANY(component_ids); /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.tokens", &names, q4, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(names) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by following %d access policies: %s", len(names), strings.Join(names, ", "))
	}

	const q5 = `SELECT name FROM columns WHERE (access_policy_id=$1 OR default_token_access_policy_id=$1) AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns */`

	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.columns", &names, q5, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(names) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by following %d columns: %s", len(names), strings.Join(names, ", "))
	}

	const q6 = `SELECT name FROM shim_object_stores WHERE access_policy_id=$1 AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns */`

	if err := s.db.SelectContext(ctx, "CheckAccessPolicyUnused.shim_object_stores", &names, q6, id); err != nil {
		return ucerr.Wrap(err)
	}

	if len(names) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "access policy is in use by following %d object stores: %s", len(names), strings.Join(names, ", "))
	}

	return nil
}
func (s Storage) preDeleteAccessPolicy(ctx context.Context, id uuid.UUID, _ /*version*/ int) error {
	// Note that we ignore the version here.
	return ucerr.Wrap(s.CheckAccessPolicyUnused(ctx, id))
}

// GetTransformerByVersion looks up a specific version of a transformer
func (s Storage) GetTransformerByVersion(ctx context.Context, id uuid.UUID, version int) (*Transformer, error) {
	const q = `
SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created
FROM transformers
WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00';`

	var tf Transformer
	if err := s.db.GetContext(ctx, "GetTransformerByVersion", &tf, q, id, version); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &tf, nil
}

// GetTransformerByName is used for de-duping to ensure names are not reused
func (s Storage) GetTransformerByName(ctx context.Context, name string) (*Transformer, error) {
	const q = `
SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created
FROM (
	SELECT a.id, a.updated, a.deleted, a.is_system, a.name, a.description, a.input_data_type_id, a.output_data_type_id, a.reuse_existing_token, a.transform_type, a.tag_ids, a.function, a.parameters, a.version, a.created
	FROM transformers AS a
	JOIN
	(select id, max(version) version FROM transformers WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
	ON a.id = b.id AND a.version=b.version) AS c
WHERE LOWER(name)=LOWER($1)
AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	var gp Transformer
	if err := s.db.GetContext(ctx, "GetTransformerByName", &gp, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &gp, nil
}

// GetTransformersForResourceIDs returns a list of transformers for the given resource ids
func (s Storage) GetTransformersForResourceIDs(ctx context.Context, errorOnMissing bool, transformerRIDs ...userstore.ResourceID) ([]Transformer, error) {
	var ids []uuid.UUID
	var names []string

	for _, rid := range transformerRIDs {
		if rid.ID != uuid.Nil {
			ids = append(ids, rid.ID)
		} else if rid.Name != "" {
			names = append(names, strings.ToLower(rid.Name))
		} else {
			return nil, ucerr.Friendlyf(nil, "invalid transformer resource id: %+v", rid)
		}
	}

	const q = `
SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created
FROM (
	SELECT a.id, a.updated, a.deleted, a.is_system, a.name, a.description, a.input_data_type_id, a.output_data_type_id, a.reuse_existing_token, a.transform_type, a.tag_ids, a.function, a.parameters, a.version, a.created
	FROM transformers AS a
	JOIN
	(select id, max(version) version FROM transformers WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
	ON a.id = b.id AND a.version=b.version) AS c
WHERE (id=ANY($1) OR LOWER(name)=ANY($2))
AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	var objects []Transformer
	if err := s.db.SelectContext(ctx, "GetTransformersForResourceIDs", &objects, q, pq.Array(ids), pq.Array(names)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	found := make([]bool, len(transformerRIDs))
	for _, obj := range objects {
		for i, rid := range transformerRIDs {
			if obj.ID == rid.ID || strings.EqualFold(obj.Name, rid.Name) {
				found[i] = true
				if rid.Name != "" && !strings.EqualFold(obj.Name, rid.Name) {
					return nil, ucerr.Errorf("transformer name mismatch for resource ID: %v, got %s", rid, obj.Name)
				}
				if rid.ID != uuid.Nil && obj.ID != rid.ID {
					return nil, ucerr.Errorf("transformer ID mismatch for resource ID: %v, got %s", rid, obj.ID)
				}
			}
		}
	}

	if errorOnMissing {
		missingRIDs := []string{}
		for i := range found {
			if !found[i] {
				missingRIDs = append(missingRIDs, fmt.Sprintf("%+v", transformerRIDs[i]))
			}
		}

		if len(missingRIDs) > 0 {
			return nil, ucerr.Friendlyf(nil, "Not all requested IDs where loaded. Missing: [%s]", strings.Join(missingRIDs, ", "))
		}
	}
	return objects, nil
}

// GetLatestTransformersPaginated returns the latest version of all transformers, paginated
func (s Storage) GetLatestTransformersPaginated(ctx context.Context, p pagination.Paginator) ([]Transformer, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf(`
SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created
FROM (
	SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created
	FROM (
			SELECT a.id, a.updated, a.deleted, a.is_system, a.name, a.description, a.input_data_type_id, a.output_data_type_id, a.reuse_existing_token, a.transform_type, a.tag_ids, a.function, a.parameters, a.version, a.created
			FROM transformers AS a
			JOIN
			(select id, max(version) version FROM transformers WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
			ON a.id = b.id AND a.version=b.version) tmp
		WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp2
	ORDER BY %s; /* lint-sql-unsafe-columns */`, p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objs []Transformer
	if err := s.db.SelectContext(ctx, "GetLatestTransformersPaginated", &objs, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	objs, respFields := pagination.ProcessResults(objs, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// GetAllTransformerVersions returns all versions of a given transformer
func (s Storage) GetAllTransformerVersions(ctx context.Context, id uuid.UUID) ([]Transformer, error) {
	const q = `SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created FROM transformers WHERE id=$1 AND deleted='0001-01-01 00:00:00';`

	var tfs []Transformer
	if err := s.db.SelectContext(ctx, "GetAllTransformerVersions", &tfs, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return tfs, nil
}

// CheckTransformerUnused checks if a transformer is in use by any accessors or token records
func (s Storage) CheckTransformerUnused(ctx context.Context, id uuid.UUID) error {
	// see if this transformer is referenced in the latest version of any accessor
	const q1 = `SELECT id FROM (
		SELECT a.id, a.deleted, a.transformer_ids FROM accessors AS a
		JOIN (select id, max(version) version
		FROM accessors
		WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
		ON a.id = b.id AND a.version=b.version) AS c
	WHERE transformer_ids @> $1::uuid[] AND deleted='0001-01-01 00:00:00'; /* lint-sql-select-partial-columns lint-sql-unsafe-columns */`

	var ids []uuid.UUID
	if err := s.db.SelectContext(ctx, "CheckTransformerUnused.accessors", &ids, q1, uuidarray.UUIDArray{id}); err != nil {
		return ucerr.Wrap(err)
	}

	if len(ids) > 0 {
		return ucerr.Friendlyf(ErrStillInUse, "transformer is in use by %d accessors", len(ids))
	}

	return nil
}

// GetTokenRecordByToken looks up a token record by the actual token, used for resolve
func (s Storage) GetTokenRecordByToken(ctx context.Context, token string) (*TokenRecord, error) {
	const q = `SELECT id, created, updated, deleted, data, token, transformer_id, transformer_version, access_policy_id, user_id, column_id FROM token_records WHERE token=$1 AND deleted='0001-01-01 00:00:00';`

	var tr TokenRecord
	if err := s.db.GetContext(ctx, "GetTokenRecordByToken", &tr, q, token); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &tr, nil
}

// ListTokenRecordsByTokens looks up token records by the actual tokens, used for resolve
func (s Storage) ListTokenRecordsByTokens(ctx context.Context, tokens []string) ([]TokenRecord, error) {
	const q = `SELECT id, created, updated, deleted, data, token, transformer_id, transformer_version, access_policy_id, user_id, column_id FROM token_records WHERE deleted='0001-01-01 00:00:00' AND token = ANY ($1)`

	var trs []TokenRecord

	if err := s.db.SelectContext(ctx, "ListTokenRecordsByTokens", &trs, q, pq.Array(&tokens)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return trs, nil
}

// ListTokenRecordsByDataAndPolicy looks up a token record by the data, transformer, and access policy id
func (s Storage) ListTokenRecordsByDataAndPolicy(ctx context.Context, data string, transformerID, accessPolicyID uuid.UUID) ([]TokenRecord, error) {
	const q = `SELECT id, created, updated, deleted, data, token, transformer_id, transformer_version, access_policy_id, user_id, column_id FROM token_records WHERE data=$1 AND transformer_id=$2 AND access_policy_id=$3 AND deleted='0001-01-01 00:00:00';`

	var trs []TokenRecord
	if err := s.db.SelectContext(ctx, "ListTokenRecordsByDataAndPolicy", &trs, q, data, transformerID, accessPolicyID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return trs, nil
}

// ListTokenRecordsByDataProvenanceAndPolicy looks up a token record by the userid/columnid reference, transformer, and access policy id
func (s Storage) ListTokenRecordsByDataProvenanceAndPolicy(ctx context.Context, userID uuid.UUID, columnID uuid.UUID, transformerID, accessPolicyID uuid.UUID) ([]TokenRecord, error) {
	const q = `SELECT id, created, updated, deleted, data, token, transformer_id, transformer_version, access_policy_id, user_id, column_id FROM token_records WHERE data=$1 AND user_id=$2 AND column_id=$3 AND transformer_id=$4 AND access_policy_id=$5 AND deleted='0001-01-01 00:00:00';`

	var trs []TokenRecord
	if err := s.db.SelectContext(ctx, "ListTokenRecordsByDataProvenanceAndPolicy", &trs, q, "", userID, columnID, transformerID, accessPolicyID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return trs, nil
}

// BatchListTokensByDataAndPolicy looks up a tokens by the data, transformers, and access policy ids
func (s Storage) BatchListTokensByDataAndPolicy(ctx context.Context, data []string, transformerIDs, accessPolicyIDs []uuid.UUID) ([]string, error) {
	if len(data) != len(transformerIDs) || len(data) != len(accessPolicyIDs) {
		return nil, ucerr.Errorf("length of data, transformerIDs, and accessPolicyIDs must match")
	}

	uniqueNames := set.NewStringSet(data...)
	uniqueAccessPolicyIDs := set.NewUUIDSet(accessPolicyIDs...)
	uniqueTransformerIDs := set.NewUUIDSet(transformerIDs...)

	// Note this query sorts in reverse order of creation time so that we always return the last token created for a given data/transformer/access policy
	const q = `SELECT id, created, updated, deleted, data, token, transformer_id, transformer_version, access_policy_id, user_id, column_id FROM token_records WHERE data=ANY($1) AND transformer_id=ANY($2) AND access_policy_id=ANY($3) AND deleted='0001-01-01 00:00:00' ORDER BY created DESC;`

	var trs []TokenRecord
	if err := s.db.SelectContext(ctx, "BatchListTokenRecordsByDataAndPolicy", &trs, q, pq.Array(uniqueNames.Items()), pq.Array(uniqueTransformerIDs.Items()), pq.Array(uniqueAccessPolicyIDs.Items())); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// this is O(n^2) but n is usually relatively small (on the order of ~100 entries)
	tokens := make([]string, len(data))
	for _, tr := range trs {
		for i, d := range data {
			if tr.Data == d && tr.AccessPolicyID == accessPolicyIDs[i] && tr.TransformerID == transformerIDs[i] && tokens[i] == "" {
				tokens[i] = tr.Token
			}
		}
	}

	return tokens, nil
}

// GetSecretByName is used for retrieving a secret by name
func (s Storage) GetSecretByName(ctx context.Context, name string) (*Secret, error) {
	const q = `SELECT created, deleted, id, name, updated, value FROM policy_secrets WHERE LOWER(name)=LOWER($1) AND deleted='0001-01-01 00:00:00';`

	var secret Secret
	if err := s.db.GetContext(ctx, "GetSecretByName", &secret, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &secret, nil
}
