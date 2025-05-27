import { APIError, JSONValue } from '@userclouds/sharedui';

export const GET_FEATURE_FLAGS_REQUEST = 'GET_FEATURE_FLAGS_REQUEST';
export const getFeatureFlagsRequest = () => ({
  type: GET_FEATURE_FLAGS_REQUEST,
});

export const GET_FEATURE_FLAGS_SUCCESS = 'GET_FEATURE_FLAGS_SUCCESS';
export const getFeatureFlagsSuccess = (response: JSONValue) => ({
  type: GET_FEATURE_FLAGS_SUCCESS,
  data: response,
});

export const GET_FEATURE_FLAGS_ERROR = 'GET_FEATURE_FLAGS_ERROR';
export const getFeatureFlagsError = (error: APIError) => ({
  type: GET_FEATURE_FLAGS_ERROR,
  data: error.message,
});
