import userStoreReducer from './userstore';
import PaginatedResult from '../models/PaginatedResult';
import { blankResourceID } from '../models/ResourceID';
import {
  Column,
  ColumnIndexType,
  blankColumnConstraints,
  NativeDataTypes,
} from '../models/TenantUserStoreConfig';
import Purpose from '../models/Purpose';
import Accessor, {
  AccessorColumn,
  AccessorSavePayload,
  columnToAccessorColumn,
  blankAccessor,
} from '../models/Accessor';
import Mutator, {
  MutatorColumn,
  columnToMutatorColumn,
  blankMutator,
} from '../models/Mutator';
import { blankPolicy } from '../models/AccessPolicy';
import {
  TOGGLE_ACCESSOR_FOR_DELETE,
  TOGGLE_ACCESSOR_LIST_EDIT_MODE,
  TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
  ADD_PURPOSE_TO_ACCESSOR,
  REMOVE_PURPOSE_FROM_ACCESSOR,
  ADD_PURPOSE_TO_ACCESSOR_TO_CREATE,
  REMOVE_PURPOSE_FROM_ACCESSOR_TO_CREATE,
} from '../actions/accessors';
import {
  TOGGLE_MUTATOR_FOR_DELETE,
  TOGGLE_MUTATOR_LIST_EDIT_MODE,
  TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
} from '../actions/mutators';
import { initialState, RootState } from '../store';
import { TOGGLE_USER_STORE_COLUMN_FOR_DELETE } from '../actions/userstore';

describe('user store reducer', () => {
  let newState: RootState;
  const myColumn: Column = {
    id: '123',
    table: 'users',
    data_type: NativeDataTypes.String,
    access_policy: blankResourceID(),
    default_transformer: blankResourceID(),
    default_token_access_policy: blankResourceID(),
    is_array: false,
    name: 'Foo_column',
    index_type: ColumnIndexType.None,
    is_system: false,
    search_indexed: false,
    constraints: blankColumnConstraints(),
  };
  const otherColumn: Column = {
    id: '456',
    table: 'users',
    data_type: NativeDataTypes.Timestamp,
    access_policy: blankResourceID(),
    default_transformer: blankResourceID(),
    default_token_access_policy: blankResourceID(),
    is_array: false,
    name: 'Bar_Column',
    index_type: ColumnIndexType.Unique,
    is_system: false,
    search_indexed: false,
    constraints: blankColumnConstraints(),
  };
  const myAccessorColumn: AccessorColumn = columnToAccessorColumn(myColumn);
  const otherAccessorColumn: AccessorColumn =
    columnToAccessorColumn(otherColumn);
  const myMutatorColumn: MutatorColumn = columnToMutatorColumn(myColumn);
  const otherMutatorColumn: MutatorColumn = columnToMutatorColumn(otherColumn);

  describe('TOGGLE_USER_STORE_COLUMN_FOR_DELETE', () => {
    it('should add a column to the delete queue', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
      };
      expect(Object.keys(state.userStoreColumnsToDelete).length).toBe(0);
      expect(state.userStoreColumnsToDelete['123']).toBeFalsy();

      newState = userStoreReducer(state, {
        type: TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
        data: myColumn,
      });

      expect(Object.keys(state.userStoreColumnsToDelete).length).toBe(1);

      expect(newState.userStoreColumnsToDelete['123'].id).toBe('123');
      expect(newState.userStoreColumnsToDelete['123'].name).toBe('Foo_column');
    });

    it('should remove a column from the queue if it was previously queued for delete', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumnsToDelete: { 456: otherColumn, 123: myColumn },
      };
      expect(state.userStoreColumnsToDelete['123']).toBeTruthy();
      expect(state.userStoreColumnsToDelete['456']).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
        data: myColumn,
      });

      expect(state.userStoreColumnsToDelete['456']).toBeTruthy();
      expect(state.userStoreColumnsToDelete['123']).toBeFalsy();

      expect(newState.userStoreColumnsToDelete['456'].id).toBe('456');
      expect(newState.userStoreColumnsToDelete['456'].name).toBe('Bar_Column');
    });

    it("should remove a column from the modified column queue when it's queued for delete", () => {
      const state: RootState = {
        ...initialState,
        userStoreColumnsToDelete: {},
        userStoreColumnsToModify: { 456: otherColumn, 123: myColumn },
      };
      expect(state.userStoreColumnsToModify['123']).toBeTruthy();
      expect(state.userStoreColumnsToModify['456']).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
        data: myColumn,
      });

      expect(state.userStoreColumnsToDelete['123']).toBeTruthy();
      expect(state.userStoreColumnsToDelete['456']).toBeFalsy();

      expect(state.userStoreColumnsToModify['123']).toBeFalsy();
      expect(state.userStoreColumnsToModify['456']).toBeTruthy();

      expect(newState.userStoreColumnsToModify['456'].id).toBe('456');
      expect(newState.userStoreColumnsToModify['456'].name).toBe('Bar_Column');

      expect(newState.userStoreColumnsToDelete['123'].id).toBe('123');
      expect(newState.userStoreColumnsToDelete['123'].name).toBe('Foo_column');
    });

    it("should remove a column from the add column queue when it's queued for delete", () => {
      const state: RootState = {
        ...initialState,
        userStoreColumnsToDelete: {},
        userStoreColumnsToAdd: [otherColumn, myColumn],
      };
      expect(state.userStoreColumnsToAdd.length).toBe(2);
      expect(state.userStoreColumnsToAdd[0].id).toBe('456');
      expect(state.userStoreColumnsToAdd[1].id).toBe('123');

      newState = userStoreReducer(state, {
        type: TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
        data: myColumn,
      });

      expect(state.userStoreColumnsToAdd.length).toBe(1);
      expect(state.userStoreColumnsToAdd[0].id).toBe('456');

      expect(state.userStoreColumnsToDelete['456']).toBeFalsy();

      expect(newState.userStoreColumnsToAdd[0].id).toBe('456');
      expect(newState.userStoreColumnsToAdd[0].name).toBe('Bar_Column');
    });
  });

  describe('TOGGLE_ACCESSOR_COLUMN_FOR_DELETE', () => {
    it('should remove an unsaved column from the add queue, without adding it to the delete queue', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedAccessor: {
          ...blankAccessor(),
          id: '123456',
          name: 'My_Accessor',
          version: 3,
          description: 'This is a simple accessor',
          columns: [otherAccessorColumn],
          access_policy: blankPolicy(),
        },
        accessorColumnsToAdd: [myAccessorColumn],
      };

      expect(state.accessorColumnsToDelete).toEqual({});
      expect(state.accessorColumnsToAdd.length).toBe(1);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
        data: myAccessorColumn,
      });

      expect(state.accessorColumnsToDelete).toEqual({});
      expect(state.accessorColumnsToAdd.length).toBe(0);
    });

    it('should add a persisted column to the delete queue', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedAccessor: {
          ...blankAccessor(),
          id: '123456',
          name: 'My_Accessor',
          version: 3,
          description: 'This is a simple accessor',
          columns: [otherAccessorColumn],
        },
      };

      expect(state.accessorColumnsToDelete).toEqual({});
      expect(state.accessorColumnsToAdd.length).toBe(0);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
        data: otherAccessorColumn,
      });

      expect(Object.keys(state.accessorColumnsToDelete).length).toBe(1);
      expect(state.accessorColumnsToDelete['456']).toBeTruthy();
      expect(state.accessorColumnsToDelete['456'].name).toBe('Bar_Column');
      expect(state.accessorColumnsToAdd.length).toBe(0);
    });

    it('should remove a persisted column from the delete queue if it is already queued', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedAccessor: {
          ...blankAccessor(),
          id: '123456',
          name: 'My_Accessor',
          version: 3,
          description: 'This is a simple accessor',
          columns: [myAccessorColumn, otherAccessorColumn],
        },
        accessorColumnsToDelete: {
          '456': otherAccessorColumn,
        },
      };

      expect(Object.keys(state.accessorColumnsToDelete).length).toBe(1);
      expect(state.accessorColumnsToDelete['456']).toBeTruthy();
      expect(state.accessorColumnsToAdd.length).toBe(0);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
        data: myAccessorColumn,
      });
      expect(Object.keys(state.accessorColumnsToDelete).length).toBe(2);
      expect(state.accessorColumnsToDelete['456']).toBeTruthy();
      expect(state.accessorColumnsToDelete['456'].name).toBe('Bar_Column');
      expect(state.accessorColumnsToDelete['123']).toBeTruthy();
      expect(state.accessorColumnsToDelete['123'].name).toBe('Foo_column');

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
        data: otherAccessorColumn,
      });

      expect(Object.keys(state.accessorColumnsToDelete).length).toBe(1);
      expect(state.accessorColumnsToDelete['456']).toBeFalsy();
      expect(state.accessorColumnsToDelete['123']).toBeTruthy();
      expect(state.accessorColumnsToDelete['123'].name).toBe('Foo_column');

      expect(state.accessorColumnsToAdd.length).toBe(0);
    });
  });

  describe('TOGGLE_MUTATOR_COLUMN_FOR_DELETE', () => {
    it('should remove an unsaved column from the add queue, without adding it to the delete queue', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedMutator: {
          ...blankMutator(),
          id: '123456',
          name: 'My_Mutator',
          version: 3,
          description: 'This is a simple mutator',
          columns: [otherMutatorColumn],
        },
        mutatorColumnsToAdd: [myMutatorColumn],
      };

      expect(state.mutatorColumnsToDelete).toEqual({});
      expect(state.mutatorColumnsToAdd.length).toBe(1);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
        data: myColumn,
      });

      expect(state.mutatorColumnsToDelete).toEqual({});
      expect(state.mutatorColumnsToAdd.length).toBe(0);
    });

    it('should add a persisted column to the delete queue', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedMutator: {
          ...blankMutator(),
          id: '123456',
          name: 'My_Mutator',
          version: 3,
          description: 'This is a simple mutator',
          columns: [otherMutatorColumn],
        },
      };

      expect(state.mutatorColumnsToDelete).toEqual({});
      expect(state.mutatorColumnsToAdd.length).toBe(0);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
        data: otherMutatorColumn,
      });

      expect(Object.keys(state.mutatorColumnsToDelete).length).toBe(1);
      expect(state.mutatorColumnsToDelete['456']).toBeTruthy();
      expect(state.mutatorColumnsToDelete['456'].name).toBe('Bar_Column');
      expect(state.mutatorColumnsToAdd.length).toBe(0);
    });

    it('should remove a persisted column from the delete queue if it is already queued', () => {
      const state: RootState = {
        ...initialState,
        userStoreColumns: [myColumn, otherColumn],
        selectedMutator: {
          ...blankMutator(),
          id: '123456',
          name: 'My_Mutator',
          version: 3,
          description: 'This is a simple mutator',
          columns: [myMutatorColumn, otherMutatorColumn],
        },
        mutatorColumnsToDelete: {
          '456': otherMutatorColumn,
        },
      };

      expect(Object.keys(state.mutatorColumnsToDelete).length).toBe(1);
      expect(state.mutatorColumnsToDelete['456']).toBeTruthy();
      expect(state.mutatorColumnsToAdd.length).toBe(0);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
        data: myMutatorColumn,
      });
      expect(Object.keys(state.mutatorColumnsToDelete).length).toBe(2);
      expect(state.mutatorColumnsToDelete['456']).toBeTruthy();
      expect(state.mutatorColumnsToDelete['456'].name).toBe('Bar_Column');
      expect(state.mutatorColumnsToDelete['123']).toBeTruthy();
      expect(state.mutatorColumnsToDelete['123'].name).toBe('Foo_column');

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
        data: otherMutatorColumn,
      });

      expect(Object.keys(state.mutatorColumnsToDelete).length).toBe(1);
      expect(state.mutatorColumnsToDelete['456']).toBeFalsy();
      expect(state.mutatorColumnsToDelete['123']).toBeTruthy();
      expect(state.mutatorColumnsToDelete['123'].name).toBe('Foo_column');

      expect(state.mutatorColumnsToAdd.length).toBe(0);
    });
  });

  describe('TOGGLE_ACCESSOR_FOR_DELETE', () => {
    const accessorOne: Accessor = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
    };
    const accessorTwo: Accessor = {
      ...blankAccessor(),
      id: '3aa4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Other_Accessor',
      description: 'This is a not so simple accessor',
    };

    it("should remove an accessor if it's already queued", () => {
      const state: RootState = {
        ...initialState,
        accessors: {
          data: [accessorOne, accessorTwo],
          has_next: false,
          next: '',
          has_prev: false,
          prev: '',
        },
        accessorsToDelete: {
          [accessorTwo.id]: accessorTwo,
        },
      };
      expect(Object.keys(state.accessorsToDelete).length).toBe(1);
      expect(state.accessorsToDelete[accessorTwo.id]).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_FOR_DELETE,
        data: accessorTwo,
      });

      expect(Object.keys(state.accessorsToDelete).length).toBe(0);
      expect(state.accessorsToDelete[accessorTwo.id]).toBeFalsy();
    });

    it("should add an accessor to the delete queue if it's not already there", () => {
      const state: RootState = {
        ...initialState,
        accessors: {
          data: [accessorOne, accessorTwo],
          has_next: false,
          next: '',
          has_prev: false,
          prev: '',
        },
        accessorsToDelete: {
          [accessorTwo.id]: accessorTwo,
        },
      };
      expect(Object.keys(state.accessorsToDelete).length).toBe(1);
      expect(state.accessorsToDelete[accessorTwo.id]).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_FOR_DELETE,
        data: accessorOne,
      });

      expect(Object.keys(state.accessorsToDelete).length).toBe(2);
      expect(state.accessorsToDelete[accessorTwo.id]).toBeTruthy();
      expect(state.accessorsToDelete[accessorOne.id]).toBeTruthy();
    });
  });

  describe('TOGGLE_ACCESSOR_LIST_EDIT_MODE', () => {
    const state: RootState = initialState;
    it('should set the edit mode variable to the value of the passed in boolean', () => {
      expect(state.accessorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_LIST_EDIT_MODE,
        data: false,
      });
      expect(newState.accessorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_LIST_EDIT_MODE,
        data: true,
      });
      expect(newState.accessorListEditMode).toBe(true);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_LIST_EDIT_MODE,
        data: false,
      });
      expect(newState.accessorListEditMode).toBe(false);
    });

    it('should set the edit mode variable to its opposite if no boolean is passed in', () => {
      expect(state.accessorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_LIST_EDIT_MODE,
      });
      expect(newState.accessorListEditMode).toBe(true);

      newState = userStoreReducer(state, {
        type: TOGGLE_ACCESSOR_LIST_EDIT_MODE,
      });
      expect(newState.accessorListEditMode).toBe(false);
    });
  });

  describe('MODIFY_ACCESSOR_ADD_PURPOSE', () => {
    const purpose1 = {
      id: 'id1',
      name: 'one',
      description: '',
      is_system: false,
    };
    const purpose2 = {
      id: 'id2',
      name: 'two',
      description: '',
      is_system: false,
    };
    const statePurposes: PaginatedResult<Purpose> = {
      data: [purpose1, purpose2],
      has_next: false,
      next: '',
      has_prev: false,
      prev: '',
    };
    const accessorOne: Accessor = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
      purposes: [purpose1],
    };
    const accessorTwo: Accessor = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
      purposes: [],
    };
    const state: RootState = {
      ...initialState,
      modifiedAccessor: accessorOne,
      purposes: statePurposes,
    };
    it('should add a purpose', () => {
      expect(state.modifiedAccessor?.purposes.length).toBe(1);
      expect(state.modifiedAccessor?.purposes[0].id).toBe('id1');
      newState = userStoreReducer(state, {
        type: ADD_PURPOSE_TO_ACCESSOR,
        data: purpose2.id,
      });
      expect(state.modifiedAccessor?.purposes.length).toBe(2);
      expect(state.modifiedAccessor?.purposes[1].id).toBe('id2');
    });
    const state1: RootState = {
      ...initialState,
      modifiedAccessor: accessorTwo,
      purposes: statePurposes,
    };
    it('should add a purpose to empty purpose array', () => {
      expect(state1.modifiedAccessor?.purposes).toEqual([]);
      newState = userStoreReducer(state1, {
        type: ADD_PURPOSE_TO_ACCESSOR,
        data: purpose2.id,
      });
      expect(state1.modifiedAccessor?.purposes.length).toBe(1);
      expect(state1.modifiedAccessor?.purposes[0].id).toBe('id2');
    });
  });

  describe('MODIFY_ACCESSOR_REMOVE_PURPOSE', () => {
    const purpose1 = {
      id: 'id1',
      name: 'one',
      description: '',
      is_system: false,
    };
    const purpose2 = {
      id: 'id2',
      name: 'two',
      description: '',
      is_system: false,
    };
    const statePurposes: PaginatedResult<Purpose> = {
      data: [purpose1, purpose2],
      has_next: false,
      next: '',
      has_prev: false,
      prev: '',
    };
    const accessorOne: Accessor = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
      purposes: [purpose1, purpose2],
    };
    const state: RootState = {
      ...initialState,
      modifiedAccessor: accessorOne,
      purposes: statePurposes,
    };
    it('should remove a purpose', () => {
      expect(state.modifiedAccessor?.purposes.length).toBe(2);
      newState = userStoreReducer(state, {
        type: REMOVE_PURPOSE_FROM_ACCESSOR,
        data: purpose2,
      });
      expect(state.modifiedAccessor?.purposes.length).toBe(1);
      expect(state.modifiedAccessor?.purposes[0].id).toBe('id1');
    });
  });

  describe('MODIFY_ACCESSOR_TO_CREATE_ADD_PURPOSE', () => {
    const purpose1 = {
      id: 'id1',
      name: 'one',
      description: '',
      is_system: false,
    };
    const purpose2 = {
      id: 'id2',
      name: 'two',
      description: '',
      is_system: false,
    };
    const statePurposes: PaginatedResult<Purpose> = {
      data: [purpose1, purpose2],
      has_next: false,
      next: '',
      has_prev: false,
      prev: '',
    };
    const accessorOne: Accessor = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
      purposes: [purpose1],
    };
    const state: RootState = {
      ...initialState,
      accessorToCreate: accessorOne,
      purposes: statePurposes,
    };
    it('should add a purpose', () => {
      expect(state.accessorToCreate?.purposes.length).toBe(1);
      expect(state.accessorToCreate?.purposes[0].id).toBe('id1');
      newState = userStoreReducer(state, {
        type: ADD_PURPOSE_TO_ACCESSOR_TO_CREATE,
        data: purpose2.id,
      });
      expect(state.accessorToCreate?.purposes.length).toBe(2);
      expect(state.accessorToCreate?.purposes[1].id).toBe('id2');
    });
  });

  describe('MODIFY_ACCESSOR_TO_CREATE_REMOVE_PURPOSE', () => {
    const purpose1 = {
      id: 'id1',
      name: 'one',
      description: '',
      is_system: false,
    };
    const purpose2 = {
      id: 'id2',
      name: 'two',
      description: '',
      is_system: false,
    };
    const accessorOne: AccessorSavePayload = {
      ...blankAccessor(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Accessor',
      description: 'This is a simple accessor',
      access_policy_id: '0c0b7371-5175-405b-a17c-fec5969914b8',
      selector_config: {
        where_clause: '{id} = ?',
      },
      purposes: [purpose1, purpose2],
    };
    const state: RootState = { ...initialState, accessorToCreate: accessorOne };
    it('should remove a purpose', () => {
      expect(state.accessorToCreate?.purposes.length).toBe(2);
      newState = userStoreReducer(state, {
        type: REMOVE_PURPOSE_FROM_ACCESSOR_TO_CREATE,
        data: purpose2,
      });
      expect(state.accessorToCreate?.purposes.length).toBe(1);
      expect(state.accessorToCreate?.purposes[0].id).toBe('id1');
    });
  });

  describe('TOGGLE_MUTATOR_FOR_DELETE', () => {
    const mutatorOne: Mutator = {
      ...blankMutator(),
      id: '2ee4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My_Mutator',
      description: 'This is a simple mutator',
    };
    const mutatorTwo = {
      ...blankMutator(),
      id: '3aa4497e-c326-4068-94ed-3dcdaaaa53bc',
      name: 'My other mutator',
      description: 'This is a not so simple mutator',
    };

    it("should remove an mutator if it's already queued", () => {
      const state: RootState = {
        ...initialState,
        mutators: {
          data: [mutatorOne, mutatorTwo],
          has_next: false,
          next: '',
          has_prev: false,
          prev: '',
        },
        mutatorsToDelete: {
          [mutatorTwo.id]: mutatorTwo,
        },
      };
      expect(Object.keys(state.mutatorsToDelete).length).toBe(1);
      expect(state.mutatorsToDelete[mutatorTwo.id]).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_FOR_DELETE,
        data: mutatorTwo,
      });

      expect(Object.keys(state.mutatorsToDelete).length).toBe(0);
      expect(state.mutatorsToDelete[mutatorTwo.id]).toBeFalsy();
    });

    it("should add an mutator to the delete queue if it's not already there", () => {
      const state: RootState = {
        ...initialState,
        mutators: {
          data: [mutatorOne, mutatorTwo],
          has_next: false,
          next: '',
          has_prev: false,
          prev: '',
        },
        mutatorsToDelete: {
          [mutatorTwo.id]: mutatorTwo,
        },
      };
      expect(Object.keys(state.mutatorsToDelete).length).toBe(1);
      expect(state.mutatorsToDelete[mutatorTwo.id]).toBeTruthy();

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_FOR_DELETE,
        data: mutatorOne,
      });

      expect(Object.keys(state.mutatorsToDelete).length).toBe(2);
      expect(state.mutatorsToDelete[mutatorTwo.id]).toBeTruthy();
      expect(state.mutatorsToDelete[mutatorOne.id]).toBeTruthy();
    });
  });

  describe('TOGGLE_MUTATOR_LIST_EDIT_MODE', () => {
    const state: RootState = initialState;
    it('should set the edit mode variable to the value of the passed in boolean', () => {
      expect(state.mutatorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
        data: false,
      });
      expect(newState.mutatorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
        data: true,
      });
      expect(newState.mutatorListEditMode).toBe(true);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
        data: false,
      });
      expect(newState.mutatorListEditMode).toBe(false);
    });

    it('should set the edit mode variable to its opposite if no boolean is passed in', () => {
      expect(state.mutatorListEditMode).toBe(false);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
      });
      expect(newState.mutatorListEditMode).toBe(true);

      newState = userStoreReducer(state, {
        type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
      });
      expect(newState.mutatorListEditMode).toBe(false);
    });
  });
});
