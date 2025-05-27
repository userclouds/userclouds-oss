export type AuthorizationRequest = {
  sourceObjectID: string;
  attribute: string;
  targetObjectID: string;
};

export type CheckAttributeResponse = {
  has_attribute: boolean;
  path: CheckAttributePathRow[];
};

export type CheckAttributePathRow = {
  object_id: string;
  edge_id: string;
};

export const getCheckAttributeRows = (
  authorizationPath: CheckAttributePathRow[]
) => {
  const rows = [];
  let sourceObject = authorizationPath[0].object_id;
  for (let i = 1; i < authorizationPath.length; i++) {
    const edge = authorizationPath[i].edge_id;
    const targetObject = authorizationPath[i].object_id;
    rows.push({
      sourceObject: sourceObject,
      edge: edge,
      targetObject: targetObject,
    });
    sourceObject = targetObject;
  }
  return rows;
};
