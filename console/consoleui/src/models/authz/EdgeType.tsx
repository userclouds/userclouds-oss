import { v4 as uuidv4 } from 'uuid';
import PaginatedResult from '../PaginatedResult';

export type Attribute = {
  id: string;
  name: string;
  direct: boolean;
  inherit: boolean;
  propagate: boolean;
};

export enum AttributeFlavors {
  direct = 'Direct',
  propagate = 'Propagate',
  inherit = 'Inherit',
}

export const getEdgeTypeFilteredByType = (
  edgeTypes: PaginatedResult<EdgeType>,
  targetObjectTypeID: string | undefined,
  sourceObjectTypeID: string | undefined
) => {
  return edgeTypes.data.filter((et) => {
    if (targetObjectTypeID && targetObjectTypeID === et.target_object_type_id) {
      return true;
    }
    if (sourceObjectTypeID && sourceObjectTypeID === et.source_object_type_id) {
      return true;
    }
    return false;
  });
};

export const changeAttributeFlavor = (attribute: Attribute, flavor: string) => {
  const newAttribute: Attribute = {
    id: attribute.id,
    name: attribute.name,
    direct: false,
    propagate: false,
    inherit: false,
  };
  if (flavor === AttributeFlavors.direct) {
    newAttribute.direct = true;
  } else if (flavor === AttributeFlavors.inherit) {
    newAttribute.inherit = true;
  } else {
    newAttribute.propagate = true;
  }
  return newAttribute;
};

export const blankEdgeType = () => {
  return {
    id: uuidv4(),
    type_name: '',
    source_object_type_id: '',
    target_object_type_id: '',
    attributes: [],
  };
};

export const blankAttribute = () => {
  return {
    id: uuidv4(),
    name: '',
    direct: true,
    inherit: false,
    propagate: false,
  };
};

export const getEdgeTypeWithoutAttribute = (
  attribute: Attribute,
  edgeType: EdgeType
) => {
  const newEdgeType: EdgeType = JSON.parse(JSON.stringify(edgeType));
  newEdgeType.attributes = newEdgeType.attributes.filter(
    (e) => e.id !== attribute.id || e.name !== attribute.name
  );
  return newEdgeType;
};

export type EdgeType = {
  id: string;
  type_name: string;
  source_object_type_id: string;
  target_object_type_id: string;
  attributes: Attribute[];
};

export const updateAttributesForEdgeType = (
  edgeType: EdgeType,
  attribute: Attribute
): EdgeType => {
  const newEdgeType = JSON.parse(JSON.stringify(edgeType));
  const i = newEdgeType.attributes.findIndex(
    (e: Attribute) => e.id === attribute.id
  );
  if (i > -1) {
    newEdgeType.attributes[i] = attribute;
  } else {
    newEdgeType.attributes.push(attribute);
  }
  return newEdgeType;
};

export const deleteAttributeFromEdgeType = (
  edgeType: EdgeType,
  attribute: Attribute
): EdgeType => {
  const newEdgeType: EdgeType = JSON.parse(JSON.stringify(edgeType));
  newEdgeType.attributes = newEdgeType.attributes.filter(
    (e) => e.id !== attribute.id || e.name !== attribute.name
  );
  return newEdgeType;
};

export const EDGE_TYPE_COLUMNS = [
  'id',
  'type_name',
  'source_object_type_id',
  'target_object_type_id',
  'created',
  'updated',
];

export const EDGE_TYPE_PREFIX = 'edge_types_';

export default EdgeType;
