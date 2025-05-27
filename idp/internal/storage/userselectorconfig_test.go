package storage_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/testhelpers"
)

type dataTypeBuilder struct {
	dataType *column.DataType
}

func newDataTypeBuilder() *dataTypeBuilder {
	dtb := &dataTypeBuilder{}
	return dtb.newDataType()
}

func (dtb *dataTypeBuilder) newDataType() *dataTypeBuilder {
	dtb.dataType = &column.DataType{
		BaseModel:          ucdb.NewBase(),
		ConcreteDataTypeID: datatype.Composite.ID,
	}
	return dtb
}

func (dtb *dataTypeBuilder) addField(dataType userstore.ResourceID, name string) *dataTypeBuilder {
	field := userstore.CompositeField{
		DataType: dataType,
		Name:     name,
	}
	dtb.dataType.CompositeAttributes.Fields = append(
		dtb.dataType.CompositeAttributes.Fields,
		column.NewCompositeFieldFromClient(field),
	)
	return dtb
}

func (dtb *dataTypeBuilder) build() *column.DataType {
	dt := dtb.dataType
	dtb.newDataType()
	return dt
}

func (dtb *dataTypeBuilder) setDescription(description string) *dataTypeBuilder {
	dtb.dataType.Description = description
	return dtb
}

func (dtb *dataTypeBuilder) setName(name string) *dataTypeBuilder {
	dtb.dataType.Name = name
	return dtb
}

type userSelectorConfigFixture struct {
	t   *testing.T
	ctx context.Context
	s   *storage.Storage
	cm  *storage.ColumnManager
	dtm *storage.DataTypeManager
}

func newUserSelectorConfigFixture(t *testing.T) (*userSelectorConfigFixture, error) {
	ctx := context.Background()
	cc, lc, ccs := testhelpers.NewTestStorage(t)
	_, ct, cdb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cc, lc)
	s := idptesthelpers.NewStorage(ctx, t, cdb, ct.ID)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tf := &userSelectorConfigFixture{
		t:   t,
		ctx: ctx,
		s:   s,
		cm:  cm,
		dtm: dtm,
	}

	cb := newColumnBuilder()

	tf.addColumn(cb.setType(datatype.CanonicalAddress.ID), "address")
	tf.addColumn(cb.setType(datatype.Boolean.ID), "bool")
	tf.addColumn(cb.setType(datatype.Integer.ID), "int")
	tf.addColumn(cb.setType(datatype.String.ID), "string")
	tf.addColumn(cb.setType(datatype.Timestamp.ID), "timestamp")
	tf.addColumn(cb.setType(datatype.UUID.ID), "uuid")
	tf.addColumn(cb.makeArray().setType(datatype.CanonicalAddress.ID), "address_array")
	tf.addColumn(cb.makeArray().setType(datatype.Boolean.ID), "bool_array")
	tf.addColumn(cb.makeArray().setType(datatype.Integer.ID), "int_array")
	tf.addColumn(cb.makeArray().setType(datatype.String.ID), "string_array")
	tf.addColumn(cb.makeArray().setType(datatype.Timestamp.ID), "timestamp_array")
	tf.addColumn(cb.makeArray().setType(datatype.UUID.ID), "uuid_array")
	tf.addColumn(cb.setType(datatype.Integer.ID).setIndexType(userstore.ColumnIndexTypeUnique), "int_unique")
	tf.addColumn(cb.setType(datatype.String.ID).setIndexType(userstore.ColumnIndexTypeUnique), "string_unique")
	tf.addColumn(cb.setType(datatype.UUID.ID).setIndexType(userstore.ColumnIndexTypeUnique), "uuid_unique")

	dataTypeID := tf.addDataType(
		newDataTypeBuilder().
			addField(datatype.String, "String").
			addField(datatype.Integer, "Int").
			addField(datatype.Boolean, "Bool"),
		"test_composite",
	)

	tf.addColumn(cb.setType(dataTypeID), "composite")
	tf.addColumn(cb.makeArray().setType(dataTypeID), "composite_array")

	return tf, nil
}

func (tf *userSelectorConfigFixture) addColumn(cb *columnBuilder, name string) uuid.UUID {
	tf.t.Helper()

	id := cb.column.ID
	cb.setName(name)
	_, err := tf.cm.SaveColumn(tf.ctx, cb.build())
	assert.NoErr(tf.t, err)
	return id
}

func (tf *userSelectorConfigFixture) addDataType(dtb *dataTypeBuilder, name string) uuid.UUID {
	tf.t.Helper()

	id := dtb.dataType.ID
	dtb.setName(name)
	dtb.setDescription("a test " + name)
	_, err := tf.dtm.SaveDataType(tf.ctx, dtb.build())
	assert.NoErr(tf.t, err)
	return id
}

func (tf *userSelectorConfigFixture) invalidSelector(clause string) {
	tf.t.Helper()
	assert.NotNil(tf.t, tf.validateSelector(column.DataLifeCycleStateLive, clause))
	assert.NotNil(tf.t, tf.validateSelector(column.DataLifeCycleStateSoftDeleted, clause))
}

func (tf *userSelectorConfigFixture) validSelector(clause string) {
	tf.t.Helper()
	assert.NoErr(tf.t, tf.validateSelector(column.DataLifeCycleStateLive, clause))
	assert.NoErr(tf.t, tf.validateSelector(column.DataLifeCycleStateSoftDeleted, clause))
}

func (tf *userSelectorConfigFixture) validateSelector(
	dlcs column.DataLifeCycleState,
	clause string,
) error {
	uscv := storage.NewUserSelectorConfigValidator(
		tf.ctx,
		tf.s,
		tf.cm,
		tf.dtm,
		dlcs,
	)
	return ucerr.Wrap(uscv.Validate(userstore.UserSelectorConfig{WhereClause: clause}))
}

func TestUserSelectorConfigSettings(t *testing.T) {
	t.Parallel()

	tf, err := newUserSelectorConfigFixture(t)
	assert.NoErr(t, err)

	// empty selector

	tf.invalidSelector("")

	// match all selector

	tf.validSelector("ALL")

	// SYSTEM columns

	tf.validSelector("{id} = ?")
	tf.validSelector("{created} = ?")
	tf.validSelector("{organization_id} = ?")
	tf.validSelector("{updated} = ?")
	tf.validSelector("{version} = ?")

	// ADDRESS column

	tf.validSelector("{address} = ?")
	tf.validSelector("{address_array} != ?")
	tf.validSelector("{address} > ?")
	tf.validSelector("{address_array} < ?")
	tf.validSelector("{address} >= ?")
	tf.validSelector("{address_array} <= ?")
	tf.validSelector("{address} IS NULL")
	tf.validSelector("{address_array} is null")
	tf.validSelector("{address} IS NOT NULL")
	tf.validSelector("{address_array} is not null")

	tf.validSelector("{address}->>'id' = ?")
	tf.validSelector("{address_array}->>'id' IS NULL")
	tf.validSelector("{address}->>'id' IS NOT NULL")
	tf.validSelector("{address_array}->>'country' = ?")
	tf.validSelector("{address}->>'name' = ?")
	tf.validSelector("{address_array}->>'organization' = ?")
	tf.validSelector("{address}->>'street_address_line_1' = ?")
	tf.validSelector("{address_array}->>'street_address_line_2' = ?")
	tf.validSelector("{address}->>'dependent_locality' = ?")
	tf.validSelector("{address_array}->>'locality' = ?")
	tf.validSelector("{address}->>'administrative_area' = ?")
	tf.validSelector("{address_array}->>'post_code' = ?")
	tf.validSelector("{address}->>'sorting_code' = ?")
	tf.validSelector("{address_array}->>'sorting_code' = 'foo'")

	tf.invalidSelector("{address_array}->>'id' = 10")
	tf.invalidSelector("{address}->>'unsupported_sub_field' = ?")

	// BOOLEAN column

	tf.validSelector("{bool} = ?")
	tf.validSelector("{bool_array} < ?")
	tf.validSelector("{bool} > ?")
	tf.validSelector("{bool_array} <= ?")
	tf.validSelector("{bool} >= ?")
	tf.validSelector("{bool_array} != ?")
	tf.validSelector("{bool} IS NULL")
	tf.validSelector("{bool_array} is null")
	tf.validSelector("{bool} IS NOT NULL")
	tf.validSelector("{bool_array} is not null")

	tf.validSelector("{bool} = 1::BOOL")
	tf.validSelector("{bool_array} = 1::BOOLEAN")
	tf.validSelector("{bool} = '1'::BOOL")
	tf.validSelector("{bool_array} = '1'::BOOLEAN")
	tf.validSelector("{bool} = 'on'")
	tf.validSelector("{bool_array} = 'on'::BOOL")
	tf.validSelector("{bool} = 'on'::BOOLEAN")
	tf.validSelector("{bool_array} = 'ON'")
	tf.validSelector("{bool} = 'ON'::BOOL")
	tf.validSelector("{bool_array} = 'ON'::BOOLEAN")
	tf.validSelector("{bool} = 't'")
	tf.validSelector("{bool_array} = 't'::BOOL")
	tf.validSelector("{bool} = 't'::BOOLEAN")
	tf.validSelector("{bool_array} = 'T'")
	tf.validSelector("{bool} = 'T'::BOOL")
	tf.validSelector("{bool_array} = 'T'::BOOLEAN")
	tf.validSelector("{bool} = true")
	tf.validSelector("{bool_array} = true::BOOL")
	tf.validSelector("{bool} = true::BOOLEAN")
	tf.validSelector("{bool_array} = 'true'")
	tf.validSelector("{bool} = 'true'::BOOL")
	tf.validSelector("{bool_array} = 'true'::BOOLEAN")
	tf.validSelector("{bool} = TRUE")
	tf.validSelector("{bool_array} = TRUE::BOOL")
	tf.validSelector("{bool} = TRUE::BOOLEAN")
	tf.validSelector("{bool_array} = 'TRUE'")
	tf.validSelector("{bool} = 'TRUE'::BOOL")
	tf.validSelector("{bool_array} = 'TRUE'::BOOLEAN")
	tf.validSelector("{bool} = 'y'")
	tf.validSelector("{bool_array} = 'y'::BOOL")
	tf.validSelector("{bool} = 'y'::BOOLEAN")
	tf.validSelector("{bool_array} = 'Y'")
	tf.validSelector("{bool} = 'Y'::BOOL")
	tf.validSelector("{bool_array} = 'Y'::BOOLEAN")
	tf.validSelector("{bool} = 'yes'")
	tf.validSelector("{bool_array} = 'yes'::BOOL")
	tf.validSelector("{bool} = 'yes'::BOOLEAN")
	tf.validSelector("{bool_array} = 'YES'")
	tf.validSelector("{bool} = 'YES'::BOOL")
	tf.validSelector("{bool_array} = 'YES'::BOOLEAN")
	tf.validSelector("{bool} = 0::BOOL")
	tf.validSelector("{bool_array} = 0::BOOLEAN")
	tf.validSelector("{bool} = '0'::BOOL")
	tf.validSelector("{bool_array} = '0'::BOOLEAN")
	tf.validSelector("{bool} = 'f'")
	tf.validSelector("{bool_array} = 'f'::BOOL")
	tf.validSelector("{bool} = 'f'::BOOLEAN")
	tf.validSelector("{bool_array} = 'F'")
	tf.validSelector("{bool} = 'F'::BOOL")
	tf.validSelector("{bool_array} = 'F'::BOOLEAN")
	tf.validSelector("{bool} = false")
	tf.validSelector("{bool_array} = false::BOOL")
	tf.validSelector("{bool} = false::BOOLEAN")
	tf.validSelector("{bool_array} = 'false'")
	tf.validSelector("{bool} = 'false'::BOOL")
	tf.validSelector("{bool_array} = 'false'::BOOLEAN")
	tf.validSelector("{bool} = FALSE")
	tf.validSelector("{bool_array} = FALSE::BOOL")
	tf.validSelector("{bool} = FALSE::BOOLEAN")
	tf.validSelector("{bool_array} = 'FALSE'")
	tf.validSelector("{bool} = 'FALSE'::BOOL")
	tf.validSelector("{bool_array} = 'FALSE'::BOOLEAN")
	tf.validSelector("{bool} = 'n'")
	tf.validSelector("{bool_array} = 'n'::BOOL")
	tf.validSelector("{bool} = 'n'::BOOLEAN")
	tf.validSelector("{bool_array} = 'N'")
	tf.validSelector("{bool} = 'N'::BOOL")
	tf.validSelector("{bool_array} = 'N'::BOOLEAN")
	tf.validSelector("{bool} = 'no'")
	tf.validSelector("{bool_array} = 'no'::BOOL")
	tf.validSelector("{bool} = 'no'::BOOLEAN")
	tf.validSelector("{bool_array} = 'NO'")
	tf.validSelector("{bool} = 'NO'::BOOL")
	tf.validSelector("{bool_array} = 'NO'::BOOLEAN")
	tf.validSelector("{bool} = 'of'")
	tf.validSelector("{bool_array} = 'of'::BOOL")
	tf.validSelector("{bool} = 'of'::BOOLEAN")
	tf.validSelector("{bool_array} = 'off'")
	tf.validSelector("{bool} = 'off'::BOOL")
	tf.validSelector("{bool_array} = 'off'::BOOLEAN")
	tf.validSelector("{bool} = 'OF'")
	tf.validSelector("{bool_array} = 'OF'::BOOL")
	tf.validSelector("{bool} = 'OF'::BOOLEAN")
	tf.validSelector("{bool_array} = 'OFF'")
	tf.validSelector("{bool} = 'OFF'::BOOL")
	tf.validSelector("{bool_array} = 'OFF'::BOOLEAN")

	tf.invalidSelector("{bool} = foo")
	tf.invalidSelector("{bool_array} = 'foo'")
	tf.invalidSelector("{bool} = 42")
	tf.invalidSelector("{bool_array} = 'N'::VARCHAR")

	// COMPOSITE column

	tf.validSelector("{composite} = ?")
	tf.validSelector("{composite_array} != ?")
	tf.validSelector("{composite} > ?")
	tf.validSelector("{composite_array} < ?")
	tf.validSelector("{composite} >= ?")
	tf.validSelector("{composite_array} <= ?")
	tf.validSelector("{composite} IS NULL")
	tf.validSelector("{composite_array} is null")
	tf.validSelector("{composite} IS NOT NULL")
	tf.validSelector("{composite_array} is not null")

	tf.validSelector("{composite}->>'string' = ?")
	tf.validSelector("{composite_array}->>'string' IS NULL")
	tf.validSelector("{composite}->>'string' IS NOT NULL")
	tf.validSelector("{composite}->>'string' = 'string'")
	tf.validSelector("{composite_array}->>'int' = 52")
	tf.validSelector("{composite_array}->>'int' = -52")
	tf.validSelector("{composite}->>'bool' = true")

	tf.invalidSelector("{composite}->>'string' = 52")
	tf.invalidSelector("{composite_array}->>'int' = 'string'")
	tf.invalidSelector("{composite}->>'bool' = 'string'")
	tf.invalidSelector("{composite}->>'unsupported_subfield' = ?")

	// INTEGER column

	tf.validSelector("{int} = ?")
	tf.validSelector("{int_array} < ?")
	tf.validSelector("{int_unique} > ?")
	tf.validSelector("{int} <= ?")
	tf.validSelector("{int_array} >= ?")
	tf.validSelector("{int_unique} != ?")
	tf.validSelector("{int} IS NULL")
	tf.validSelector("{int_array} is null")
	tf.validSelector("{int_unique} IS NOT NULL")
	tf.validSelector("{int} is not null")

	tf.validSelector("{int} = 2")
	tf.validSelector("{int_array} = 2::INT")
	tf.validSelector("{int_unique} = 2::INTEGER")
	tf.validSelector("{int} = +2")
	tf.validSelector("{int_array} = +2::INT")
	tf.validSelector("{int_unique} = +2::INTEGER")
	tf.validSelector("{int} = -2")
	tf.validSelector("{int_array} = -2::INT")
	tf.validSelector("{int_unique} = -2::INTEGER")
	tf.validSelector("{int} = '2'")
	tf.validSelector("{int_array} = '2'::INT")
	tf.validSelector("{int_unique} = '2'::INTEGER")
	tf.validSelector("{int} = '+2'")
	tf.validSelector("{int_array} = '+2'::INT")
	tf.validSelector("{int_unique} = '+2'::INTEGER")
	tf.validSelector("{int} = '-2'")
	tf.validSelector("{int_array} = '-2'::INT")
	tf.validSelector("{int_unique} = '-2'::INTEGER")

	tf.validSelector("abs({int_array}) = ?")
	tf.validSelector("ABS({int_unique}) = 4")
	tf.validSelector("mod({int},2) = ?")
	tf.validSelector("MOD({int_array},?) = ?")
	tf.validSelector("mod({int_unique},?) = 2")
	tf.validSelector("MOD({int},2) = 1")
	tf.validSelector("MOD({int_array},2::INTEGER) = 1::INTEGER")
	tf.validSelector("div({int_unique},3) = ?")
	tf.validSelector("div({int_unique},-3) = ?")
	tf.validSelector("DIV({int},?) = ?")
	tf.validSelector("div({int_array},?) = 2")
	tf.validSelector("DIV({int_unique},2::INTEGER) = 1::INTEGER")

	tf.validSelector("div(abs({int}), 2) = ?")
	tf.validSelector("div(mod({int_array}, 2), 2) = ?")
	tf.validSelector("abs(div(mod({int_unique}, 2), 2)) = ?")

	tf.invalidSelector("{int} = 'foo'")
	tf.invalidSelector("{int} = '10'::VARCHAR")
	tf.invalidSelector("abs(div(mod({string}, 2), 2)) = ?")

	// STRING column

	tf.validSelector("{string} = ?")
	tf.validSelector("{string_array} < ?")
	tf.validSelector("{string_unique} > ?")
	tf.validSelector("{string} <= ?")
	tf.validSelector("{string_array} >= ?")
	tf.validSelector("{string_unique} != ?")
	tf.validSelector("{string} IS NULL")
	tf.validSelector("{string_array} is null")
	tf.validSelector("{string_unique} IS NOT NULL")
	tf.validSelector("{string} is not null")

	tf.validSelector("{string} = 'foo'")
	tf.validSelector("{string_array} = 'fo''o'")
	tf.validSelector("{string_unique} = 'foo'::VARCHAR")

	tf.validSelector("{string} like ?")
	tf.validSelector("{string_array} LIKE ?")
	tf.validSelector("{string_unique} ilike ?")
	tf.validSelector("{string} ILIKE ?")
	tf.validSelector("{string_array} like 'foo'")
	tf.validSelector("{string_unique} ilike 'foo'")

	tf.validSelector("char_length({string}) = ?")
	tf.validSelector("CHAR_LENGTH({string_array}) = 4")
	tf.validSelector("character_length({string_unique}) = ?")
	tf.validSelector("CHARACTER_LENGTH({string}) = ?")
	tf.validSelector("lower({string_array}) = ?")
	tf.validSelector("LOWER({string_unique}) = 'foo'")
	tf.validSelector("UPPER({string}) = ?")
	tf.validSelector("upper({string_array}) = 'FOO'")

	tf.validSelector("lower(upper({string_unique})) = ?")
	tf.validSelector("char_length(upper({string})) = ?")
	tf.validSelector("char_length(upper({string_array})) = 4")

	tf.invalidSelector("{string} = 4")
	tf.invalidSelector("{string} = false")
	tf.invalidSelector("char_length({string}) = 'foo'")
	tf.invalidSelector("lower({string}) = 4")
	tf.invalidSelector("upper({string}) = 4")

	// TIMESTAMP column

	tf.validSelector("{timestamp} = ?")
	tf.validSelector("{timestamp_array} < ?")
	tf.validSelector("{timestamp} > ?")
	tf.validSelector("{timestamp_array} <= ?")
	tf.validSelector("{timestamp} >= ?")
	tf.validSelector("{timestamp_array} != ?")
	tf.validSelector("{timestamp} IS NULL")
	tf.validSelector("{timestamp_array} is null")
	tf.validSelector("{timestamp} IS NOT NULL")
	tf.validSelector("{timestamp_array} is not null")

	tf.validSelector("{timestamp_array} = '2024-01-01 00:00:00'")
	tf.validSelector("{timestamp} = '2024-01-01 00:00:00'::TIMESTAMP")

	tf.validSelector("DATE_PART(?,{timestamp_array}) = ?")
	tf.validSelector("date_part('day',{timestamp}) = ?")
	tf.validSelector("DATE_PART(?,{timestamp_array}) = 10")
	tf.validSelector("date_part('day',{timestamp}) = 10")
	tf.validSelector("DATE_PART('dow',{timestamp_array}) = ?")
	tf.validSelector("date_part('epoch',{timestamp}) = ?")
	tf.validSelector("DATE_PART('hour',{timestamp_array}) = ?")
	tf.validSelector("date_part('microseconds',{timestamp}) = ?")
	tf.validSelector("DATE_PART('milliseconds',{timestamp_array}) = ?")
	tf.validSelector("date_part('minute',{timestamp}) = ?")
	tf.validSelector("DATE_PART('month',{timestamp_array}) = ?")
	tf.validSelector("date_part('second',{timestamp}) = ?")
	tf.validSelector("DATE_PART('timezone',{timestamp_array}) = ?")
	tf.validSelector("date_part('week',{timestamp}) = ?")
	tf.validSelector("DATE_PART('year',{timestamp_array}) = ?")

	tf.validSelector("DATE_TRUNC(?,{timestamp}) = ?")
	tf.validSelector("date_trunc('day',{timestamp_array}) = ?")
	tf.validSelector("DATE_TRUNC(?,{timestamp}) = '2024-01-01 00:00:00'")
	tf.validSelector("date_trunc('day',{timestamp_array}) = '2024-01-01 00:00:00'::TIMESTAMP")
	tf.validSelector("DATE_TRUNC('hour',{timestamp}) = ?")
	tf.validSelector("date_trunc('microseconds',{timestamp_array}) = ?")
	tf.validSelector("DATE_TRUNC('milliseconds',{timestamp}) = ?")
	tf.validSelector("date_trunc('minute',{timestamp_array}) = ?")
	tf.validSelector("DATE_TRUNC('month',{timestamp}) = ?")
	tf.validSelector("date_trunc('second',{timestamp_array}) = ?")
	tf.validSelector("DATE_TRUNC('week',{timestamp}) = ?")
	tf.validSelector("date_trunc('year',{timestamp_array}) = ?")

	tf.validSelector("date_part('year', DATE_TRUNC('hour',{timestamp})) = ?")
	tf.validSelector("date_part('year', date_trunc('hour',{timestamp_array})) = 2024")
	tf.validSelector("date_part('year', DATE_TRUNC('hour',{timestamp})) = 2024::INTEGER")

	tf.invalidSelector("{timestamp} = 'foo'")
	tf.invalidSelector("{timestamp} = 42")
	tf.invalidSelector("date_part('foo',{timestamp}) = ?")
	tf.invalidSelector("date_part('year', date_part('hour',{timestamp})) = ?")
	tf.invalidSelector("date_trunc('foo',{timestamp}) = ?")
	tf.invalidSelector("date_trunc('year', date_part('hour',{timestamp})) = ?")

	// UUID column

	tf.validSelector("{uuid} = ?")
	tf.validSelector("{uuid_array} < ?")
	tf.validSelector("{uuid_unique} > ?")
	tf.validSelector("{uuid} <= ?")
	tf.validSelector("{uuid_array} >= ?")
	tf.validSelector("{uuid_unique} != ?")
	tf.validSelector("{uuid} IS NULL")
	tf.validSelector("{uuid_array} is null")
	tf.validSelector("{uuid_unique} IS NOT NULL")
	tf.validSelector("{uuid} is not null")

	tf.validSelector("{uuid_array} = '00000000-0000-0000-0000-000000000000'")
	tf.validSelector("{uuid_unique} = '00000000-0000-0000-0000-000000000000'::UUID")

	tf.invalidSelector("{uuid} = '00000000-0000-0000-0000-00000000'")
	tf.invalidSelector("{uuid_array} = 'foo'")
	tf.invalidSelector("{uuid_unique} = 42")

	// ANY operator

	tf.validSelector("{int} = ANY (?)")
	tf.validSelector("{int} = any ((?))")
	tf.validSelector("{int} = ANY (ARRAY[1])")
	tf.validSelector("{int} = ANY (array[1,2,4,5])")
	tf.validSelector("{int} = ANY (ARRAY[1,2,4,5])")
	tf.validSelector("{int} = any (array[1,2::INT,4::INTEGER,'5','6'::INT,?,'7'::INTEGER,?])")
	tf.validSelector("{int} = ANY (ARRAY [ 1, 2, 4, 5 ])")

	tf.validSelector("{bool} = ANY (ARRAY[true,false,?,'T'::BOOL,'FALSE'::BOOLEAN])")

	tf.validSelector("{string} = ANY (( ARRAY['10',?,'bar'::VARCHAR, 'string''with''single-quotes'::VARCHAR] ) )")

	tf.invalidSelector("{int} = ARRAY[1]")
	tf.invalidSelector("{int} = ANY ARRAY['foo']")
	tf.invalidSelector("{bool} = ANY ARRAY['foo']")
	tf.invalidSelector("{string} = ANY ARRAY[1]")

	// parentheses and spacing

	tf.validSelector("{id} = (?)")
	tf.validSelector("({id}=?)")
	tf.validSelector("({id}=(?))")
	tf.validSelector("{id}<?")
	tf.validSelector("{id}>?")
	tf.validSelector("{id}<=?")
	tf.validSelector("{id}>=?")
	tf.validSelector("{id}!=?")

	// conjunctions

	tf.validSelector("{id} = ? OR {id} = ?")
	tf.validSelector("{id} = ? or {id} = ?")
	tf.validSelector("{id} = ? AND {id} = ?")
	tf.validSelector("{id} = ? and {id} = ?")
	tf.validSelector("{id} = ? OR {id} = ? OR {id} = ?")
	tf.validSelector("{id} = ? AND {id} = ? AND {id} = ?")
	tf.validSelector("{id} = ? OR {id} = ? AND {id} = ?")
	tf.validSelector("( {id} = ? OR {id} = ?) AND {id} = ?")
	tf.validSelector("{id} = ? OR ({id} = ? AND {id} = ? )")

	// badly formed selectors

	tf.invalidSelector("({id}) = ?")
	tf.invalidSelector("{id} = = ?")
	tf.invalidSelector("{id} = ? = ?")
	tf.invalidSelector("? = {id}")
	tf.invalidSelector("? = ?")
	tf.invalidSelector("(?) = ANY (?)")
	tf.invalidSelector("{id} = ? ?")
	tf.invalidSelector("{id} = = = (?)")
	tf.invalidSelector("{id} LIKE LIKE")
	tf.invalidSelector("{id} >= <=")
	tf.invalidSelector("{id} ANY (?)")
	tf.invalidSelector("{id} ANY ANY (?)")
	tf.invalidSelector("{id} = ? OR ")
	tf.invalidSelector("{id}LIKE ?")
	tf.invalidSelector("{id} LIKE?")
	tf.invalidSelector("{id}")
	tf.invalidSelector("columnX = (?)")
	tf.invalidSelector("{json_column}->>'asdf' IS NOT NOT NULL")
	tf.invalidSelector("{json_column}->>'asdf' = NULL")

	// incompatible operators

	tf.invalidSelector("abs({address}) = ?")
	tf.invalidSelector("abs({bool}) = ?")
	tf.invalidSelector("abs({composite}) = ?")
	tf.invalidSelector("abs({string}) = ?")
	tf.invalidSelector("abs({timestamp}) = ?")
	tf.invalidSelector("abs({uuid}) = ?")

	tf.invalidSelector("char_length({address}) = ?")
	tf.invalidSelector("char_length({bool}) = ?")
	tf.invalidSelector("char_length({composite}) = ?")
	tf.invalidSelector("char_length({int}) = ?")
	tf.invalidSelector("char_length({timestamp}) = ?")
	tf.invalidSelector("char_length({uuid}) = ?")

	tf.invalidSelector("date_part('day',{address}) = ?")
	tf.invalidSelector("date_part('day',{bool}) = ?")
	tf.invalidSelector("date_part('day',{composite}) = ?")
	tf.invalidSelector("date_part('day',{int}) = ?")
	tf.invalidSelector("date_part('day',{string}) = ?")
	tf.invalidSelector("date_part('day',{uuid}) = ?")

	tf.invalidSelector("date_trunc('day',{address}) = ?")
	tf.invalidSelector("date_trunc('day',{bool}) = ?")
	tf.invalidSelector("date_trunc('day',{composite}) = ?")
	tf.invalidSelector("date_trunc('day',{int}) = ?")
	tf.invalidSelector("date_trunc('day',{string}) = ?")
	tf.invalidSelector("date_trunc('day',{uuid}) = ?")

	tf.invalidSelector("div({address}, 1) = ?")
	tf.invalidSelector("div({bool}, 1) = ?")
	tf.invalidSelector("div({composite}, 1) = ?")
	tf.invalidSelector("div({string}, 1) = ?")
	tf.invalidSelector("div({timestamp}, 1) = ?")
	tf.invalidSelector("div({uuid}, 1) = ?")

	tf.invalidSelector("{address} ilike ?")
	tf.invalidSelector("{bool} ilike ?")
	tf.invalidSelector("{composite} ilike ?")
	tf.invalidSelector("{int} ilike ?")
	tf.invalidSelector("{timestamp} ilike ?")
	tf.invalidSelector("{uuid} ilike ?")

	tf.invalidSelector("{address} like ?")
	tf.invalidSelector("{bool} like ?")
	tf.invalidSelector("{composite} like ?")
	tf.invalidSelector("{int} like ?")
	tf.invalidSelector("{timestamp} like ?")
	tf.invalidSelector("{uuid} like ?")

	tf.invalidSelector("lower({address}) = ?")
	tf.invalidSelector("lower({bool}) = ?")
	tf.invalidSelector("lower({composite}) = ?")
	tf.invalidSelector("lower({int}) = ?")
	tf.invalidSelector("lower({timestamp}) = ?")
	tf.invalidSelector("lower({uuid}) = ?")

	tf.invalidSelector("mod({address}, 1) = ?")
	tf.invalidSelector("mod({bool}, 1) = ?")
	tf.invalidSelector("mod({composite}, 1) = ?")
	tf.invalidSelector("mod({string}, 1) = ?")
	tf.invalidSelector("mod({timestamp}, 1) = ?")
	tf.invalidSelector("mod({uuid}, 1) = ?")

	tf.invalidSelector("upper({address}) = ?")
	tf.invalidSelector("upper({bool}) = ?")
	tf.invalidSelector("upper({composite}) = ?")
	tf.invalidSelector("upper({int}) = ?")
	tf.invalidSelector("upper({timestamp}) = ?")
	tf.invalidSelector("upper({uuid}) = ?")
}
