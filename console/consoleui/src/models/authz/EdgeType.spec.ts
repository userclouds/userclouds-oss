import EdgeType, {
  Attribute,
  deleteAttributeFromEdgeType,
  getEdgeTypeWithoutAttribute,
  updateAttributesForEdgeType,
} from './EdgeType';

const attribute1: Attribute = {
  id: 'a1',
  name: 'testA1',
  direct: false,
  inherit: false,
  propagate: false,
};

const attribute1_updated: Attribute = {
  id: 'a1',
  name: 'testA1Updated',
  direct: false,
  inherit: false,
  propagate: false,
};

const attribute2: Attribute = {
  id: 'a2',
  name: 'testA2',
  direct: false,
  inherit: false,
  propagate: false,
};

const attribute3: Attribute = {
  id: 'a3',
  name: 'testA3',
  direct: false,
  inherit: false,
  propagate: false,
};

const edgeType1: EdgeType = {
  id: 'et1',
  type_name: 'testET1',
  source_object_type_id: 'ot1',
  target_object_type_id: 'ot2',
  attributes: [attribute1, attribute2],
};

const edgeType2: EdgeType = {
  id: 'et2',
  type_name: 'testET2',
  source_object_type_id: 'ot1',
  target_object_type_id: 'ot2',
  attributes: [],
};

const edgeType3: EdgeType = {
  id: 'et3',
  type_name: 'testET3',
  source_object_type_id: 'ot1',
  target_object_type_id: 'ot2',
  attributes: [attribute1, attribute1],
};

describe('getEdgeTypeWithoutAttribute', () => {
  it('get an EdgeType without attribute', () => {
    const updatedEdgeType1 = getEdgeTypeWithoutAttribute(attribute1, edgeType1);
    expect(updatedEdgeType1.attributes.length).toBe(1);
  });

  it('get an EdgeType without attributes from an EdgeType with no attributes', () => {
    const updatedEdgeType2 = getEdgeTypeWithoutAttribute(attribute1, edgeType2);
    expect(updatedEdgeType2.attributes.length).toBe(0);
  });

  it('get an EdgeType without attributes from an EdgeType with duplicate attributes', () => {
    const updatedEdgeType3 = getEdgeTypeWithoutAttribute(attribute1, edgeType3);
    expect(updatedEdgeType3.attributes.length).toBe(0);
  });
});

describe('updateAttributesForEdgeType', () => {
  it('update an EdgeType with a matching attribute', () => {
    const newEdgeType1 = updateAttributesForEdgeType(
      edgeType1,
      attribute1_updated
    );
    expect(newEdgeType1.attributes.length).toBe(2);
    expect(newEdgeType1.id).toBe('et1');
    expect(newEdgeType1.attributes[0].name).toBe('testA1Updated');
    expect(newEdgeType1.attributes[0].id).toBe('a1');
  });

  it('add an attribute for an edgetype with no attributes', () => {
    const newEdgeType2 = updateAttributesForEdgeType(edgeType2, attribute1);
    expect(newEdgeType2.attributes.length).toBe(1);
    expect(newEdgeType2.id).toBe('et2');
    expect(newEdgeType2.attributes[0].name).toBe('testA1');
    expect(newEdgeType2.attributes[0].id).toBe('a1');
  });

  it('add an attribute for an edgetype with no matching attributes', () => {
    const newEdgeType3 = updateAttributesForEdgeType(edgeType1, attribute3);
    expect(newEdgeType3.attributes.length).toBe(3);
  });
});

describe('deleteAttributeFromEdgeType', () => {
  it('delete from an EdgeType with a matching attribute', () => {
    const newEdgeType1 = deleteAttributeFromEdgeType(edgeType1, attribute1);
    expect(newEdgeType1.attributes.length).toBe(1);
  });

  it('not delete an attribute for an edgetype with no attributes', () => {
    const newEdgeType2 = deleteAttributeFromEdgeType(edgeType2, attribute1);
    expect(newEdgeType2.attributes.length).toBe(0);
  });

  it('not delete an attribute for an edgetype with no matching attributes', () => {
    const newEdgeType3 = deleteAttributeFromEdgeType(edgeType1, attribute3);
    expect(newEdgeType3.attributes.length).toBe(2);
  });
});
