import {
  CheckAttributePathRow,
  getCheckAttributeRows,
} from '../models/authz/CheckAttribute';

describe('getCheckAttributeRows', () => {
  it('should create one row per edge - 1', () => {
    const path: CheckAttributePathRow[] = [
      { object_id: '123', edge_id: 'e123' },
      { object_id: '456', edge_id: 'e456' },
    ];

    const rows = getCheckAttributeRows(path);
    expect(rows.length).toBe(1);
    expect(rows[0].edge).toBe('e456');
    expect(rows[0].sourceObject).toBe('123');
    expect(rows[0].targetObject).toBe('456');
  });
  it('should create one row per edge - 1 for longer paths', () => {
    const path: CheckAttributePathRow[] = [
      { object_id: '123', edge_id: 'e123' },
      { object_id: '456', edge_id: 'e456' },
      { object_id: '789', edge_id: 'e789' },
      { object_id: '101', edge_id: 'e101' },
    ];

    const rows = getCheckAttributeRows(path);
    expect(rows.length).toBe(3);
    expect(rows[0].edge).toBe('e456');
    expect(rows[0].sourceObject).toBe('123');
    expect(rows[0].targetObject).toBe('456');
    expect(rows[1].edge).toBe('e789');
    expect(rows[1].sourceObject).toBe('456');
    expect(rows[1].targetObject).toBe('789');
    expect(rows[2].edge).toBe('e101');
    expect(rows[2].sourceObject).toBe('789');
    expect(rows[2].targetObject).toBe('101');
  });
});
