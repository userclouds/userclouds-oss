import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  CREATE_TAG_SUCCESS,
  CREATE_TAG_ERROR,
  CREATE_TAG_REQUEST,
  FETCH_TAGS_REQUEST,
  FETCH_TAGS_ERROR,
  FETCH_TAGS_SUCCESS,
} from '../actions/tags';

const tagReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case FETCH_TAGS_REQUEST:
      state.fetchingTags = true;
      state.fetchTagsError = '';
      break;
    case FETCH_TAGS_SUCCESS:
      state.tags = action.data;
      state.fetchingTags = false;
      break;
    case FETCH_TAGS_ERROR:
      state.fetchingTags = false;
      state.fetchTagsError = action.data;
      break;
    case CREATE_TAG_REQUEST:
      state.savingTags = true;
      state.savingTagsError = '';
      break;
    case CREATE_TAG_SUCCESS:
      state.savingTags = false;
      break;
    case CREATE_TAG_ERROR:
      state.savingTags = false;
      state.savingTagsError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default tagReducer;
