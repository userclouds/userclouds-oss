import authzReducer from './authz';
import { initialState, RootState } from '../store';
import {
  CheckAttributePathRow,
  CheckAttributeResponse,
} from '../models/authz/CheckAttribute';
import { CHECK_AUTHORIZATION_RESULT } from '../actions/authz';

describe('authz reducer', () => {
  let newState: RootState;
  describe('CHECK_AUTHORIZATION_RESULT', () => {
    it('should add a path if result has the attribute', () => {
      const path: CheckAttributePathRow[] = [
        { object_id: '123', edge_id: 'e123' },
        { object_id: '456', edge_id: 'e456' },
      ];

      const state: RootState = { ...initialState, authorizationPath: path };
      expect(Object.keys(state.authorizationPath).length).toBe(2);

      const response: CheckAttributeResponse = {
        has_attribute: true,
        path: path,
      };

      newState = authzReducer(state, {
        type: CHECK_AUTHORIZATION_RESULT,
        data: response,
      });

      expect(newState.authorizationPath.length).toBe(2);
      expect(newState.authorizationPath[0].object_id).toBe('123');
      expect(newState.authorizationPath[0].edge_id).toBe('e123');
      expect(newState.authorizationPath[1].object_id).toBe('456');
      expect(newState.authorizationPath[1].edge_id).toBe('e456');
    });
    it('should not add a path if result does not have the attribute', () => {
      const path: CheckAttributePathRow[] = [
        { object_id: '123', edge_id: 'e123' },
        { object_id: '456', edge_id: 'e456' },
      ];

      const state: RootState = { ...initialState, authorizationPath: path };
      expect(Object.keys(state.authorizationPath).length).toBe(2);

      const response: CheckAttributeResponse = {
        has_attribute: false,
        path: path,
      };

      newState = authzReducer(state, {
        type: CHECK_AUTHORIZATION_RESULT,
        data: response,
      });

      expect(newState.authorizationPath.length).toBe(0);
    });
  });
});
