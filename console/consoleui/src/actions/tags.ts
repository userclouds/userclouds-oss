import { APIError } from '@userclouds/sharedui';

import { TagModel } from '../models/Tag';
import { AppDispatch } from '../store';

export const CREATE_TAG_REQUEST = 'CREATE_TAG_REQUEST';
export const CREATE_TAG_SUCCESS = 'CREATE_TAG_SUCCESS';
export const CREATE_TAG_ERROR = 'CREATE_TAG_ERROR';

export const FETCH_TAGS_REQUEST = 'FETCH_TAGS_REQUEST';
export const FETCH_TAGS_SUCCESS = 'FETCH_TAGS_SUCCESS';
export const FETCH_TAGS_ERROR = 'FETCH_TAGS_ERROR';

export const fetchTagsRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_TAGS_REQUEST,
  });
};

export const fetchTagsSuccess =
  (tags: TagModel[]) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_TAGS_SUCCESS,
      data: tags,
    });
  };

export const fetchTagsError = (error: APIError) => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_TAGS_ERROR,
    data: error.message,
  });
};

export const createTagRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_TAG_REQUEST,
  });
};

export const createTagSuccess =
  (tag: TagModel[]) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_TAG_SUCCESS,
      data: tag,
    });
  };

export const createTagError = (error: APIError) => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_TAG_ERROR,
    data: error.message,
  });
};
