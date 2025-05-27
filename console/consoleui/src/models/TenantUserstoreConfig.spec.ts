import { CompositeAttributes, CompositeField, DataType } from './DataType';
import { blankResourceID, ResourceID } from './ResourceID';
import {
  blankColumnConstraints,
  ColumnConstraints,
  Column,
  ColumnIndexType,
  uniqueIDsAvailable,
  immutableAvailable,
  partialUpdatesAvailable,
  uniqueValuesAvailable,
  NativeDataTypes,
} from './TenantUserStoreConfig';

const compositeFieldString: CompositeField = {
  data_type: NativeDataTypes.String,
  name: 'field1',
  camel_case_name: 'Field1',
  struct_name: 'field_1',
  required: false,
  ignore_for_uniqueness: false,
};
const compositeFieldInteger: CompositeField = {
  data_type: NativeDataTypes.Integer,
  name: 'field2',
  camel_case_name: 'Field2',
  struct_name: 'field_2',
  required: false,
  ignore_for_uniqueness: false,
};
const compositeAttributes1: CompositeAttributes = {
  include_id: false,
  fields: [compositeFieldString, compositeFieldInteger],
};

describe('uniqueIDsAvailable', () => {
  const datatype1: DataType = {
    id: 'id1',
    name: 'name1',
    description: 'desc1',
    is_composite_field_type: true,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID1: ResourceID = {
    id: 'id1',
    name: 'name1',
  };
  const datatype2: DataType = {
    id: 'id2',
    name: 'name2',
    description: 'desc2',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID2: ResourceID = {
    id: 'id2',
    name: 'name2',
  };
  const datatypes: DataType[] = [datatype1, datatype2];

  it('should pass when data type exists and is composite', () => {
    const column1: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID1,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueIDsAvailable(column1, datatypes)).toBeTruthy();
  });
  it('should fail when data type is not composite', () => {
    const column2: Column = {
      id: 'id2',
      table: 'users',
      name: 'name2',
      data_type: datatypeID2,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueIDsAvailable(column2, datatypes)).toBeFalsy();
  });
  it('should fail when data type is is not found', () => {
    const column3: Column = {
      id: 'id3',
      table: 'users',
      name: 'name3',
      data_type: blankResourceID(),
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueIDsAvailable(column3, datatypes)).toBeFalsy();
  });
});

describe('immutableAvailable', () => {
  const datatype1: DataType = {
    id: 'id1',
    name: 'name1',
    description: 'desc1',
    is_composite_field_type: true,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID1: ResourceID = {
    id: 'id1',
    name: 'name1',
  };

  const datatypes: DataType[] = [datatype1];

  const constraints1: ColumnConstraints = {
    immutable_required: false,
    partial_updates: false,
    unique_id_required: true,
    unique_required: true,
  };

  it('should pass when data type is composite and unique ids are available and unique is required', () => {
    const column1: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID1,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints1,
    };
    expect(immutableAvailable(column1, datatypes)).toBeTruthy();
  });
  it('should fail when data type is not composite and unique ids are not available and unique is required', () => {
    const column2: Column = {
      id: 'id2',
      table: 'users',
      name: 'name2',
      data_type: NativeDataTypes.Integer,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints1,
    };

    expect(immutableAvailable(column2, datatypes)).toBeFalsy();
  });
  it('should fail when data type is composite and unique ids are available and unique is not required', () => {
    const column3: Column = {
      id: 'id3',
      table: 'users',
      name: 'name3',
      data_type: datatypeID1,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(immutableAvailable(column3, datatypes)).toBeFalsy();
  });
  it('should fail when data type is not composite and unique ids are not available and unique is not required', () => {
    const column4: Column = {
      id: 'id4',
      table: 'users',
      name: 'name4',
      data_type: NativeDataTypes.Integer,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(immutableAvailable(column4, datatypes)).toBeFalsy();
  });
});

describe('partialUpdatesAvailable', () => {
  const constraints1: ColumnConstraints = {
    immutable_required: false,
    partial_updates: false,
    unique_id_required: true,
    unique_required: true,
  };

  it('should pass when column is array and unique id is required and unique is required', () => {
    const column1: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: NativeDataTypes.String,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: true,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints1,
    };

    expect(partialUpdatesAvailable(column1)).toBeTruthy();
  });
  it('should pass when column is array and unique id is not required and unique is required', () => {
    const constraints2: ColumnConstraints = {
      immutable_required: false,
      partial_updates: false,
      unique_id_required: false,
      unique_required: true,
    };
    const column2: Column = {
      id: 'id2',
      table: 'users',
      name: 'name2',
      data_type: NativeDataTypes.String,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: true,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints2,
    };
    expect(partialUpdatesAvailable(column2)).toBeTruthy();
  });
  it('should pass when column is array and unique id is required and unique is not required', () => {
    const constraints3: ColumnConstraints = {
      immutable_required: false,
      partial_updates: false,
      unique_id_required: true,
      unique_required: false,
    };
    const column3: Column = {
      id: 'id3',
      table: 'users',
      name: 'name3',
      data_type: NativeDataTypes.String,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: true,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints3,
    };

    expect(partialUpdatesAvailable(column3)).toBeTruthy();
  });
  it('should fail when column is not array and unique id is required and unique is required', () => {
    const constraints4: ColumnConstraints = {
      immutable_required: false,
      partial_updates: false,
      unique_id_required: false,
      unique_required: false,
    };
    const column4: Column = {
      id: 'id4',
      table: 'users',
      name: 'name4',
      data_type: NativeDataTypes.String,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints4,
    };

    expect(partialUpdatesAvailable(column4)).toBeFalsy();
  });
  it('should fail when column is not array and unique id is required and unique is required', () => {
    const column5: Column = {
      id: 'id5',
      table: 'users',
      name: 'name5',
      data_type: NativeDataTypes.String,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: constraints1,
    };
    expect(partialUpdatesAvailable(column5)).toBeFalsy();
  });
});

describe('uniqueValuesAvailable', () => {
  const datatype1: DataType = {
    id: NativeDataTypes.Integer.id,
    name: NativeDataTypes.Integer.name,
    description: 'desc1',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID1: ResourceID = {
    id: NativeDataTypes.Integer.id,
    name: NativeDataTypes.Integer.name,
  };
  const datatype2: DataType = {
    id: 'id2',
    name: 'name2',
    description: 'desc2',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID2: ResourceID = {
    id: 'id2',
    name: 'name2',
  };

  const datatype3: DataType = {
    id: 'id3',
    name: 'name3',
    description: 'desc3',
    is_composite_field_type: true,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID3: ResourceID = {
    id: 'id3',
    name: 'name3',
  };

  const datatype4: DataType = {
    id: 'id4',
    name: 'name4',
    description: 'desc4',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: compositeAttributes1,
  };
  const datatypeID4: ResourceID = {
    id: 'id4',
    name: 'name4',
  };

  const compositeField1: CompositeField = {
    data_type: NativeDataTypes.Integer,
    name: 'integer',
    camel_case_name: 'Name',
    struct_name: 'integer',
    required: true,
    ignore_for_uniqueness: false,
  };

  const datatype5: DataType = {
    id: 'id5',
    name: 'name5',
    description: 'desc5',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: {
      include_id: false,
      fields: [compositeField1],
    },
  };
  const datatypeID5: ResourceID = {
    id: 'id5',
    name: 'name5',
  };

  const compositeField2: CompositeField = {
    data_type: datatypeID2,
    name: '',
    camel_case_name: '',
    struct_name: '',
    required: true,
    ignore_for_uniqueness: false,
  };

  const datatype6: DataType = {
    id: 'id6',
    name: 'name6',
    description: 'desc6',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: {
      include_id: false,
      fields: [compositeField1, compositeField2],
    },
  };
  const datatypeID6: ResourceID = {
    id: 'id6',
    name: 'name6',
  };

  const datatype7: DataType = {
    id: 'id7',
    name: 'name7',
    description: 'desc7',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: {
      include_id: false,
      fields: [compositeField2],
    },
  };
  const datatypeID7: ResourceID = {
    id: 'id7',
    name: 'name7',
  };

  const datatype8: DataType = {
    id: 'id8',
    name: 'name8',
    description: 'desc8',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: {
      include_id: true,
      fields: [compositeField1],
    },
  };
  const datatypeID8: ResourceID = {
    id: 'id8',
    name: 'name8',
  };

  const datatypes: DataType[] = [
    datatype1,
    datatype2,
    datatype3,
    datatype4,
    datatype5,
    datatype6,
    datatype7,
    datatype8,
  ];

  const column1: Column = {
    id: 'id1',
    table: 'users',
    name: 'name1',
    data_type: datatypeID1,
    access_policy: blankResourceID(),
    default_transformer: blankResourceID(),
    default_token_access_policy: blankResourceID(),
    is_array: false,
    index_type: ColumnIndexType.None,
    is_system: false,
    search_indexed: false,
    constraints: blankColumnConstraints(),
  };

  it('should fail when data type is not found', () => {
    expect(uniqueValuesAvailable(column1, [])).toBeFalsy();
  });

  it('should pass when data type is found and is not composite and is allowable type', () => {
    expect(uniqueValuesAvailable(column1, datatypes)).toBeTruthy();
  });

  it('should not pass when data type is found and is not composite and is allowable type', () => {
    const column2: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID2,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueValuesAvailable(column2, datatypes)).toBeFalsy();
  });

  it('should not pass when data type is found and is composite and is not allowable type', () => {
    const column3: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID3,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueValuesAvailable(column3, datatypes)).toBeFalsy();
  });

  it('should not pass when data type is found and is not composite and is not allowable type', () => {
    const column4: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID4,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };

    expect(uniqueValuesAvailable(column4, datatypes)).toBeFalsy();
  });

  it('should pass when data type composite attributes does not include id and fields is the correct length and right type', () => {
    const column5: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID5,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };

    expect(uniqueValuesAvailable(column5, datatypes)).toBeTruthy();
  });
  it('should not pass when data type composite attributes does not include id and fields is the wrong length and right type', () => {
    const column6: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID6,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };

    expect(uniqueValuesAvailable(column6, datatypes)).toBeFalsy();
  });
  it('should not pass when data type composite attributes does not include id and fields is the wrong length and right typr', () => {
    const column7: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID7,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };

    expect(uniqueValuesAvailable(column7, datatypes)).toBeFalsy();
  });
  it('should not pass when data type composite attributes and does include id and fields is the correct length and right type', () => {
    const column8: Column = {
      id: 'id1',
      table: 'users',
      name: 'name1',
      data_type: datatypeID8,
      access_policy: blankResourceID(),
      default_transformer: blankResourceID(),
      default_token_access_policy: blankResourceID(),
      is_array: false,
      index_type: ColumnIndexType.None,
      is_system: false,
      search_indexed: false,
      constraints: blankColumnConstraints(),
    };
    expect(uniqueValuesAvailable(column8, datatypes)).toBeFalsy();
  });
});
