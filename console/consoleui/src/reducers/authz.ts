import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  blankEdgeType,
  deleteAttributeFromEdgeType,
  updateAttributesForEdgeType,
} from '../models/authz/EdgeType';
import { blankEdge } from '../models/authz/Edge';
import { blankObjectType } from '../models/authz/ObjectType';
import { blankObject } from '../models/authz/Object';

import {
  CHANGE_CURRENT_OBJECT_SEARCH_FILTER,
  CHANGE_CURRENT_EDGE_SEARCH_FILTER,
  BULK_UPDATE_OBJECTS_END,
  BULK_UPDATE_OBJECTS_START,
  DELETE_OBJECT_ERROR,
  DELETE_OBJECT_SUCCESS,
  TOGGLE_OBJECT_FOR_DELETE,
  CHANGE_OBJECT,
  RETRIEVE_BLANK_OBJECT,
  GET_OBJECT_SUCCESS,
  CREATE_OBJECT_ERROR,
  CREATE_OBJECT_SUCCESS,
  CREATE_OBJECT_REQUEST,
  CHANGE_EDGE,
  RETRIEVE_BLANK_EDGE,
  GET_EDGE_ERROR,
  GET_EDGE_REQUEST,
  BULK_UPDATE_EDGES_END,
  BULK_UPDATE_EDGES_START,
  BULK_UPDATE_EDGE_TYPES_END,
  BULK_UPDATE_EDGE_TYPES_START,
  BULK_UPDATE_OBJECT_TYPES_END,
  BULK_UPDATE_OBJECT_TYPES_START,
  CHANGE_AUTHORIZATION_REQUEST,
  CHANGE_EDGE_TYPE,
  CHANGE_EDGE_TYPE_ATTRIBUTE,
  CHANGE_EDGE_TYPE_DELETE_ATTRIBUTE,
  CHANGE_OBJECT_TYPE,
  CHECK_AUTHORIZATION_ERROR,
  CHECK_AUTHORIZATION_REQUEST,
  CHECK_AUTHORIZATION_RESULT,
  CREATE_EDGE_ERROR,
  CREATE_EDGE_REQUEST,
  CREATE_EDGE_SUCCESS,
  CREATE_EDGE_TYPE_ERROR,
  CREATE_EDGE_TYPE_REQUEST,
  CREATE_EDGE_TYPE_SUCCESS,
  CREATE_OBJECT_TYPE_ERROR,
  CREATE_OBJECT_TYPE_REQUEST,
  CREATE_OBJECT_TYPE_SUCCESS,
  DELETE_EDGE_ERROR,
  DELETE_EDGE_SUCCESS,
  DELETE_EDGE_TYPES_ERROR,
  DELETE_EDGE_TYPES_SUCCESS,
  DELETE_OBJECT_TYPES_ERROR,
  DELETE_OBJECT_TYPES_SUCCESS,
  GET_DISPLAY_EDGES_ERROR,
  GET_DISPLAY_EDGES_REQUEST,
  GET_DISPLAY_EDGES_SUCCESS,
  GET_DISPLAY_OBJECTS_ERROR,
  GET_DISPLAY_OBJECTS_REQUEST,
  GET_DISPLAY_OBJECTS_SUCCESS,
  GET_EDGES_REQUEST,
  GET_EDGES_SUCCESS,
  GET_EDGE_SUCCESS,
  GET_EDGE_TYPES_ERROR,
  GET_EDGE_TYPES_REQUEST,
  GET_EDGE_TYPES_SUCCESS,
  GET_EDGE_TYPE_ERROR,
  GET_EDGE_TYPE_REQUEST,
  GET_EDGE_TYPE_SUCCESS,
  GET_OBJECT_ERROR,
  GET_OBJECT_REQUEST,
  GET_OBJECT_TYPES_ERROR,
  GET_OBJECT_TYPES_REQUEST,
  GET_OBJECT_TYPES_SUCCESS,
  GET_OBJECT_TYPE_ERROR,
  GET_OBJECT_TYPE_REQUEST,
  GET_OBJECT_TYPE_SUCCESS,
  RETRIEVE_BLANK_EDGE_TYPE,
  RETRIEVE_BLANK_OBJECT_TYPE,
  TOGGLE_EDGE_EDIT_MODE,
  TOGGLE_EDGE_FOR_DELETE,
  TOGGLE_EDGE_TYPE_EDIT_MODE,
  TOGGLE_EDGE_TYPE_FOR_DELETE,
  TOGGLE_SELECT_ALL_EDGES,
  TOGGLE_OBJECT_TYPE_EDIT_MODE,
  TOGGLE_OBJECT_TYPE_FOR_DELETE,
  UPDATE_EDGE_TYPE_ERROR,
  UPDATE_EDGE_TYPE_REQUEST,
  UPDATE_EDGE_TYPE_SUCCESS,
  UPDATE_OBJECT_TYPE_ERROR,
  UPDATE_OBJECT_TYPE_SUCCESS,
  CHANGE_CURRENT_OBJECT_TYPE_SEARCH_FILTER,
  CHANGE_CURRENT_EDGE_TYPE_SEARCH_FILTER,
  TOGGLE_SELECT_ALL_OBJECT_TYPES,
  TOGGLE_SELECT_ALL_OBJECTS,
  TOGGLE_SELECT_ALL_EDGE_TYPES,
} from '../actions/authz';
import { getNewToggleEditValue } from './reducerHelper';
import { setOperatorsForFilter } from '../controls/SearchHelper';

const authzReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case CHANGE_CURRENT_EDGE_SEARCH_FILTER:
      state.currentEdgeSearchFilter = {
        ...state.currentEdgeSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case CHANGE_CURRENT_OBJECT_SEARCH_FILTER:
      state.currentObjectSearchFilter = {
        ...state.currentObjectSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case CHANGE_CURRENT_OBJECT_TYPE_SEARCH_FILTER:
      state.objectTypeSearchFilter = {
        ...state.objectTypeSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case CHANGE_CURRENT_EDGE_TYPE_SEARCH_FILTER:
      state.edgeTypeSearchFilter = {
        ...state.edgeTypeSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case CHANGE_AUTHORIZATION_REQUEST: {
      const { sourceObjectID, attribute, targetObjectID } = action.data;
      state.authorizationRequest = {
        sourceObjectID,
        attribute,
        targetObjectID,
      };
      state.authorizationFailure = '';
      state.authorizationPath = [];
      state.authorizationSuccess = '';
      break;
    }
    case CHECK_AUTHORIZATION_REQUEST:
      state.authorizationFailure = '';
      state.authorizationPath = [];
      state.authorizationSuccess = '';
      break;
    case CHECK_AUTHORIZATION_RESULT:
      if (action.data) {
        if (action.data.has_attribute) {
          if (state.authorizationRequest) {
            state.authorizationSuccess = `Source Object (${state.authorizationRequest.sourceObjectID}) is connected by
            Attribute (${state.authorizationRequest.attribute}) to Target Object (${state.authorizationRequest.targetObjectID}).`;
          } else {
            state.authorizationSuccess = 'A path exists.';
          }
          state.authorizationPath = action.data.path;
          state.authorizationFailure = '';
        } else {
          state.authorizationSuccess = '';
          state.authorizationPath = [];
          state.authorizationFailure = 'No path exists.';
        }
      } else {
        state.authorizationSuccess = '';
        state.authorizationFailure = '';
        state.authorizationPath = [];
      }
      break;
    case CHECK_AUTHORIZATION_ERROR:
      state.authorizationFailure = action.data;
      state.authorizationPath = [];
      state.authorizationSuccess = '';
      break;
    case GET_OBJECT_TYPES_REQUEST:
      state.objectTypes = undefined;
      state.fetchObjectTypesError = '';
      state.fetchingObjectTypes = true;
      break;
    case GET_OBJECT_TYPES_SUCCESS:
      state.objectTypes = action.data;
      state.fetchingObjectTypes = false;
      break;
    case GET_OBJECT_TYPES_ERROR:
      state.fetchObjectTypesError = action.data;
      state.fetchingObjectTypes = false;
      break;
    case CREATE_OBJECT_TYPE_REQUEST:
      state.savingObjectType = true;
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = '';
      break;
    case CREATE_OBJECT_TYPE_SUCCESS:
      state.savingObjectType = false;
      state.saveObjectTypeSuccess = 'Successfully Saved';
      state.saveObjectTypeError = '';
      break;
    case CREATE_OBJECT_TYPE_ERROR:
      state.savingObjectType = false;
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = action.data;
      break;
    case UPDATE_OBJECT_TYPE_SUCCESS:
      state.savingObjectType = false;
      state.saveObjectTypeSuccess = 'Successfully Saved';
      state.saveObjectTypeError = '';
      state.currentObjectType = blankObjectType();
      break;
    case UPDATE_OBJECT_TYPE_ERROR:
      state.savingObjectType = false;
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = action.data;
      break;
    case TOGGLE_OBJECT_TYPE_EDIT_MODE:
      state.objectTypeEditMode = getNewToggleEditValue(
        action.data,
        state.objectTypeEditMode
      );
      break;
    case TOGGLE_SELECT_ALL_OBJECT_TYPES:
      if (state.objectTypes?.data?.length) {
        const shouldMarkForDelete =
          state.objectTypeDeleteQueue.length === 0 ||
          (state.objectTypeDeleteQueue.length > 0 &&
            state.objectTypeDeleteQueue.length < state.objectTypes.data.length);
        if (shouldMarkForDelete) {
          state.objectTypeDeleteQueue = state.objectTypes.data.map(
            (ot) => ot.id
          );
        } else {
          state.objectTypeDeleteQueue = [];
        }
      }
      break;
    case DELETE_OBJECT_TYPES_SUCCESS:
      state.objectTypeDeleteSuccess = action.data[0];
      state.objectTypeDeleteError = '';
      state.objectTypeDeleteQueue = [
        ...state.objectTypeDeleteQueue.filter((id) => id !== action.data),
      ];
      break;
    case DELETE_OBJECT_TYPES_ERROR:
      state.objectTypeDeleteSuccess = '';
      state.objectTypeDeleteError = action.data[0];
      break;
    case GET_OBJECT_TYPE_REQUEST:
      state.fetchObjectTypeError = '';
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = '';
      state.objectTypeDeleteError = '';
      state.objectDeleteSuccess = '';
      break;
    case GET_OBJECT_TYPE_SUCCESS:
      state.currentObjectType = action.data;
      state.fetchObjectTypeError = '';
      break;
    case GET_OBJECT_TYPE_ERROR:
      state.fetchObjectTypeError = action.data;
      break;
    case RETRIEVE_BLANK_OBJECT_TYPE:
      state.currentObjectType = blankObjectType();
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = '';
      state.objectTypeDeleteError = '';
      state.objectDeleteSuccess = '';
      break;
    case CHANGE_OBJECT_TYPE:
      state.currentObjectType = {
        ...state.currentObjectType,
        ...action.data,
      };
      state.saveObjectTypeSuccess = '';
      state.saveObjectTypeError = '';
      break;
    case TOGGLE_OBJECT_TYPE_FOR_DELETE:
      if (state.objectTypeDeleteQueue.includes(action.data)) {
        state.objectTypeDeleteQueue = state.objectTypeDeleteQueue.filter(
          (id: string) => id !== action.data
        );
      } else {
        state.objectTypeDeleteQueue = [
          ...state.objectTypeDeleteQueue,
          action.data,
        ];
      }
      break;
    case BULK_UPDATE_OBJECT_TYPES_START:
      state.savingObjectTypes = true;
      break;
    case BULK_UPDATE_OBJECT_TYPES_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.objectTypeEditMode = false;
      }
      state.savingObjectTypes = false;
      break;
    case CREATE_EDGE_TYPE_REQUEST:
      state.savingEdgeType = true;
      state.saveEdgeTypeSuccess = '';
      state.saveEdgeTypeError = '';
      break;
    case CREATE_EDGE_TYPE_SUCCESS:
      state.savingEdgeType = false;
      state.saveEdgeTypeSuccess = 'Successfully Saved';
      state.saveEdgeTypeError = '';
      state.currentEdgeType = action.data;
      state.edgeTypeIsDirty = false;
      state.edgeTypeEditMode = false;
      break;
    case CREATE_EDGE_TYPE_ERROR:
      state.savingEdgeType = false;
      state.saveEdgeTypeSuccess = '';
      state.saveEdgeTypeError = action.data;
      break;
    case UPDATE_EDGE_TYPE_REQUEST:
      state.savingEdgeType = true;
      state.saveEdgeTypeSuccess = '';
      state.saveEdgeTypeError = '';
      break;
    case UPDATE_EDGE_TYPE_SUCCESS:
      state.savingEdgeType = false;
      state.saveEdgeTypeSuccess = 'Successfully Saved';
      state.saveEdgeTypeError = '';
      state.edgeTypeEditMode = false;
      break;
    case UPDATE_EDGE_TYPE_ERROR:
      state.savingEdgeType = false;
      state.saveEdgeTypeSuccess = '';
      state.saveEdgeTypeError = action.data;
      break;
    case TOGGLE_EDGE_TYPE_EDIT_MODE:
      state.edgeTypeEditMode = getNewToggleEditValue(
        action.data,
        state.edgeTypeEditMode
      );
      state.edgeTypeDeleteQueue = [];
      state.edgeTypeDeleteError = '';
      state.edgeTypeDeleteSuccess = '';
      state.edgeTypeFetchError = '';
      state.saveEdgeTypeError = '';
      state.saveEdgeTypeSuccess = '';
      break;
    case GET_EDGE_TYPES_REQUEST:
      state.edgeTypes = undefined;
      state.fetchEdgeTypesError = '';
      state.edgeTypeDeleteSuccess = '';
      state.fetchingEdgeTypes = true;
      break;
    case DELETE_EDGE_TYPES_SUCCESS:
      state.edgeTypeDeleteSuccess = action.data[0];
      state.edgeTypeDeleteError = '';
      state.edgeTypeDeleteQueue = [
        ...state.edgeTypeDeleteQueue.filter((id) => id !== action.data),
      ];
      break;
    case DELETE_EDGE_TYPES_ERROR:
      state.edgeTypeDeleteSuccess = '';
      state.edgeTypeDeleteError = action.data[0];
      break;
    case GET_EDGE_TYPES_SUCCESS:
      state.edgeTypes = action.data;
      state.fetchingEdgeTypes = false;
      break;
    case GET_EDGE_TYPES_ERROR:
      state.fetchEdgeTypesError = action.data;
      state.fetchingEdgeTypes = false;
      break;
    case GET_EDGE_TYPE_REQUEST:
      state.fetchEdgeTypeError = '';
      state.edgeTypeEditMode = false;
      break;
    case GET_EDGE_TYPE_SUCCESS:
      state.currentEdgeType = action.data;
      if (state.currentEdgeType && !state.currentEdgeType?.attributes) {
        state.currentEdgeType.attributes = [];
      }
      state.edgeTypeIsDirty = false;
      state.fetchEdgeTypeError = '';
      break;
    case GET_EDGE_TYPE_ERROR:
      state.fetchEdgeTypeError = action.data;
      break;
    case RETRIEVE_BLANK_EDGE_TYPE:
      state.currentEdgeType = blankEdgeType();
      state.edgeTypeIsDirty = false;
      state.edgeTypeEditMode = true;
      if (state.objectTypes) {
        state.currentEdgeType.target_object_type_id =
          state.objectTypes.data[0].id;
        state.currentEdgeType.source_object_type_id =
          state.objectTypes.data[0].id;
      }
      break;
    case CHANGE_EDGE_TYPE:
      state.currentEdgeType = {
        ...state.currentEdgeType,
        ...action.data,
      };
      state.saveEdgeTypeSuccess = '';
      state.saveEdgeTypeError = '';
      state.edgeTypeIsDirty = true;
      break;
    case CHANGE_EDGE_TYPE_ATTRIBUTE:
      if (state.currentEdgeType) {
        state.currentEdgeType = JSON.parse(
          JSON.stringify(
            updateAttributesForEdgeType(state.currentEdgeType, action.data)
          )
        );
        state.saveEdgeTypeSuccess = '';
        state.saveEdgeTypeError = '';
        state.edgeTypeIsDirty = true;
      }
      break;
    case CHANGE_EDGE_TYPE_DELETE_ATTRIBUTE:
      if (state.currentEdgeType) {
        state.currentEdgeType = {
          ...deleteAttributeFromEdgeType(state.currentEdgeType, action.data),
        };
        state.saveEdgeTypeSuccess = '';
        state.saveEdgeTypeError = '';
        state.edgeTypeIsDirty = true;
      }
      break;
    case TOGGLE_EDGE_TYPE_FOR_DELETE:
      if (state.edgeTypeDeleteQueue.includes(action.data)) {
        state.edgeTypeDeleteQueue = state.edgeTypeDeleteQueue.filter(
          (id: string) => id !== action.data
        );
      } else {
        state.edgeTypeDeleteQueue = [...state.edgeTypeDeleteQueue, action.data];
      }
      break;
    case TOGGLE_SELECT_ALL_EDGES:
      if (state.displayEdges?.data?.length) {
        const shouldMarkForDelete =
          state.edgeDeleteQueue.length === 0 ||
          (state.edgeDeleteQueue.length > 0 &&
            state.edgeDeleteQueue.length < state.displayEdges.data.length);
        if (shouldMarkForDelete) {
          state.edgeDeleteQueue = state.displayEdges.data.map((ot) => ot.id);
        } else {
          state.edgeDeleteQueue = [];
        }
      }
      break;
    case BULK_UPDATE_EDGE_TYPES_START:
      state.savingEdgeTypes = true;
      break;
    case BULK_UPDATE_EDGE_TYPES_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.edgeTypeEditMode = false;
      }
      state.savingEdgeTypes = false;
      break;
    case TOGGLE_SELECT_ALL_EDGE_TYPES:
      if (state.edgeTypes?.data?.length) {
        const shouldMarkForDelete =
          state.edgeTypeDeleteQueue.length === 0 ||
          (state.edgeTypeDeleteQueue.length > 0 &&
            state.edgeTypeDeleteQueue.length < state.edgeTypes.data.length);
        if (shouldMarkForDelete) {
          state.edgeTypeDeleteQueue = state.edgeTypes.data.map((ot) => ot.id);
        } else {
          state.edgeTypeDeleteQueue = [];
        }
      }
      break;
    case GET_DISPLAY_OBJECTS_REQUEST:
      state.fetchObjectsError = '';
      state.fetchingAuthzObjects = true;
      break;
    case GET_DISPLAY_OBJECTS_SUCCESS:
      state.fetchingAuthzObjects = false;
      state.authzObjects = action.data;
      break;
    case GET_DISPLAY_OBJECTS_ERROR:
      state.authzObjects = undefined;
      state.fetchObjectsError = action.data;
      state.fetchingAuthzObjects = false;
      break;
    case GET_OBJECT_REQUEST:
      state.currentObject = undefined;
      state.saveObjectError = '';
      state.saveObjectSuccess = '';
      state.fetchObjectError = '';
      break;
    case GET_OBJECT_ERROR:
      state.fetchObjectError = action.data;
      break;
    case TOGGLE_SELECT_ALL_OBJECTS:
      if (state.authzObjects?.data?.length) {
        const shouldMarkForDelete =
          state.objectDeleteQueue.length === 0 ||
          (state.objectDeleteQueue.length > 0 &&
            state.objectDeleteQueue.length < state.authzObjects.data.length);
        if (shouldMarkForDelete) {
          state.objectDeleteQueue = state.authzObjects.data.map((ot) => ot.id);
        } else {
          state.objectDeleteQueue = [];
        }
      }
      break;
    case GET_EDGES_REQUEST:
      state.edgesForObject = undefined;
      state.fetchEdgesForObjectError = '';
      break;
    case GET_DISPLAY_EDGES_REQUEST:
      state.fetchEdgesError = '';
      state.fetchingAuthzEdges = true;
      break;
    case GET_DISPLAY_EDGES_SUCCESS:
      state.fetchingAuthzObjects = false;
      state.fetchingAuthzEdges = false;
      state.displayEdges = action.data;
      break;
    case GET_DISPLAY_EDGES_ERROR:
      state.displayEdges = undefined;
      state.fetchEdgesError = action.data;
      state.fetchingAuthzEdges = false;
      break;
    case TOGGLE_EDGE_EDIT_MODE:
      state.edgeEditMode = !state.edgeEditMode;
      break;
    case TOGGLE_EDGE_FOR_DELETE:
      if (
        state.edgeDeleteQueue &&
        state.edgeDeleteQueue.indexOf(action.data) > -1
      ) {
        state.edgeDeleteQueue = state.edgeDeleteQueue.filter(
          (id) => action.data !== id
        );
      } else {
        state.edgeDeleteQueue = [...state.edgeDeleteQueue, action.data];
      }
      break;
    case BULK_UPDATE_EDGES_START:
      state.savingEdges = true;
      break;
    case BULK_UPDATE_EDGES_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.edgeEditMode = false;
      }
      state.savingEdges = false;
      break;
    case DELETE_EDGE_SUCCESS:
      state.editEdgesSuccess = 'Successfully deleted: ' + action.data;
      state.editEdgesError = '';
      state.edgeDeleteQueue = state.edgeDeleteQueue.filter(
        (id) => id !== action.data
      );
      break;
    case DELETE_EDGE_ERROR:
      state.editEdgesSuccess = '';
      state.editEdgesError = 'Error deleting: ' + action.data;
      state.edgeDeleteQueue = state.edgeDeleteQueue.filter(
        (id) => id !== action.data
      );
      break;
    case GET_EDGES_SUCCESS:
      state.edgesForObject = action.data.edgesForObject;
      state.objectsForEdgesForObject = action.data.objectsForEdgesForObject;
      state.fetchEdgesForObjectError = action.data.fetchEdgesForObjectError;
      break;
    case CREATE_EDGE_REQUEST:
      state.savingEdge = true;
      state.saveEdgeSuccess = '';
      state.saveEdgeError = '';
      break;
    case CREATE_EDGE_SUCCESS:
      state.savingEdge = false;
      state.saveEdgeSuccess = 'Successfully Saved';
      state.saveEdgeError = '';
      state.currentEdge = blankEdge();
      state.edgeIsDirty = false;
      if (state.edgeTypes) {
        state.currentEdge.edge_type_id = state.edgeTypes.data[0].id;
      }
      break;
    case CREATE_EDGE_ERROR:
      state.savingEdge = false;
      state.saveEdgeSuccess = '';
      state.saveEdgeError = action.data;
      break;
    case GET_EDGE_SUCCESS:
      state.currentEdge = action.data;
      state.edgeIsDirty = false;
      state.fetchEdgeError = '';
      break;
    case GET_EDGE_REQUEST:
      state.currentEdge = undefined;
      state.saveEdgeError = '';
      state.saveEdgeSuccess = '';
      state.fetchEdgeError = '';
      break;
    case GET_EDGE_ERROR:
      state.fetchEdgeError = action.data;
      break;
    case RETRIEVE_BLANK_EDGE:
      state.currentEdge = blankEdge();
      state.saveEdgeSuccess = '';
      state.saveEdgeError = '';
      state.edgeIsDirty = false;
      if (state.edgeTypes) {
        state.currentEdge.edge_type_id = state.edgeTypes.data[0].id;
      }
      break;
    case CHANGE_EDGE:
      state.currentEdge = {
        id: action.data.id,
        edge_type_id: action.data.edge_type_id,
        source_object_id: action.data.source_object_id,
        target_object_id: action.data.target_object_id,
      };
      state.saveEdgeSuccess = '';
      state.saveEdgeError = '';
      state.edgeIsDirty = true;
      break;
    case CREATE_OBJECT_REQUEST:
      state.savingObject = true;
      state.editObjectsSuccess = '';
      state.saveObjectSuccess = '';
      state.saveObjectError = '';
      break;
    case CREATE_OBJECT_SUCCESS:
      state.savingObject = false;
      state.saveObjectSuccess = 'Successfully Saved';
      state.saveObjectError = '';
      state.currentObject = blankObject();
      state.objectIsDirty = false;
      if (state.objectTypes) {
        state.currentObject.type_id = state.objectTypes.data[0].id;
      }
      break;
    case CREATE_OBJECT_ERROR:
      state.savingObject = false;
      state.saveObjectSuccess = '';
      state.saveObjectError = action.data;
      break;
    case GET_OBJECT_SUCCESS:
      state.currentObject = { ...action.data };
      state.objectIsDirty = false;
      state.fetchObjectError = '';
      break;
    case RETRIEVE_BLANK_OBJECT:
      state.currentObject = blankObject();
      state.saveObjectSuccess = '';
      state.saveObjectError = '';
      state.objectIsDirty = false;
      if (state.objectTypes) {
        state.currentObject.type_id = state.objectTypes.data[0].id;
      }
      break;
    case CHANGE_OBJECT:
      state.currentObject = { ...state.currentObject, ...action.data };
      state.saveObjectSuccess = '';
      state.editObjectsSuccess = '';
      state.saveObjectError = '';
      state.objectIsDirty = true;
      break;
    case TOGGLE_OBJECT_FOR_DELETE:
      if (
        state.objectDeleteQueue &&
        state.objectDeleteQueue.indexOf(action.data) > -1
      ) {
        state.objectDeleteQueue = state.objectDeleteQueue.filter(
          (id) => action.data !== id
        );
      } else {
        state.objectDeleteQueue = [...state.objectDeleteQueue, action.data];
      }
      break;
    case DELETE_OBJECT_SUCCESS:
      state.editObjectsSuccess = 'Successfully deleted: ' + action.data;
      state.editObjectsError = '';
      state.objectDeleteQueue = state.objectDeleteQueue.filter(
        (id) => id !== action.data
      );
      break;
    case DELETE_OBJECT_ERROR:
      state.editObjectsSuccess = '';
      state.editObjectsError = 'Error deleting: ' + action.data;
      state.objectDeleteQueue = state.objectDeleteQueue.filter(
        (id) => id !== action.data
      );
      break;
    case BULK_UPDATE_OBJECTS_START:
      state.savingObjects = true;
      break;
    case BULK_UPDATE_OBJECTS_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.editingObjects = false;
      }
      state.savingObjects = false;
      break;
    default:
      break;
  }

  return state;
};

export default authzReducer;
