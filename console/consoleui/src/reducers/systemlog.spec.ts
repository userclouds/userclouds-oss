import systemLogReducer from './systemlog';
import { initialState, RootState } from '../store';
import {
  CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER,
  CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER,
} from '../actions/systemLog';

describe('system log reducer', () => {
  let newState: RootState;
  describe('CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER', () => {
    it('should handle operators for alias correctly', () => {
      const state: RootState = initialState;

      const alias = 'test text';

      newState = systemLogReducer(state, {
        type: CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER,
        data: {
          columnName: 'alias',
          value: alias,
        },
      });

      expect(newState.systemLogSearchFilter.columnName).toBe('alias');
      expect(newState.systemLogSearchFilter.operator).toBe('LK');
      expect(newState.systemLogSearchFilter.value).toBe(alias);
    });

    it('should handle date operators correctly', () => {
      const state: RootState = initialState;

      const date1 = '12/5';
      const date2 = '12/6';

      newState = systemLogReducer(state, {
        type: CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER,
        data: {
          columnName: 'created',
          value: date1,
          value2: date2,
        },
      });

      expect(newState.systemLogSearchFilter.columnName).toBe('created');
      expect(newState.systemLogSearchFilter.operator).toBe('GE');
      expect(newState.systemLogSearchFilter.value).toBe(date1);
      expect(newState.systemLogSearchFilter.operator2).toBe('LE');
      expect(newState.systemLogSearchFilter.value2).toBe(date2);
    });
  });
  describe('CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER', () => {
    it('should handle operators for alias correctly', () => {
      const state: RootState = initialState;

      const alias = 'test text';

      newState = systemLogReducer(state, {
        type: CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER,
        data: {
          columnName: 'alias',
          value: alias,
        },
      });

      expect(newState.systemLogEntryDetailSearchFilter.columnName).toBe(
        'alias'
      );
      expect(newState.systemLogEntryDetailSearchFilter.operator).toBe('LK');
      expect(newState.systemLogEntryDetailSearchFilter.value).toBe(alias);
    });

    it('should handle date operators correctly', () => {
      const state: RootState = initialState;

      const date1 = '12/5';
      const date2 = '12/6';

      newState = systemLogReducer(state, {
        type: CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER,
        data: {
          columnName: 'created',
          value: date1,
          value2: date2,
        },
      });

      expect(newState.systemLogEntryDetailSearchFilter.columnName).toBe(
        'created'
      );
      expect(newState.systemLogEntryDetailSearchFilter.operator).toBe('GE');
      expect(newState.systemLogEntryDetailSearchFilter.value).toBe(date1);
      expect(newState.systemLogEntryDetailSearchFilter.operator2).toBe('LE');
      expect(newState.systemLogEntryDetailSearchFilter.value2).toBe(date2);
    });
  });
});
