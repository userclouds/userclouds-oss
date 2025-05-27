import {
  getFilteredTags,
  getFilteredTagsAndSearch,
  getTagNameFromID,
} from './Tag';

const systemTags = [
  {
    id: 'da0a0adb-5895-418e-8961-1f0fdf69c27b',
    name: 'authentication / purpose',
  },
  { id: 'd07d0c43-8e98-4c6f-aa3c-db508fbbcbf0', name: 'geo / ip' },
  { id: '180f24bf-8984-44c9-94ec-75a872fadd38', name: 'marketing' },
  {
    id: '9bf4ef17-5a4d-4964-81c8-267a603aa520',
    name: 'marketing / advertising',
  },
  { id: '99fa7b28-7830-4980-aa65-68461419dade', name: 'marketing / direct' },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e540',
    name: 'marketing / direct / email',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e541',
    name: 'marketing / direct / email1',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e542',
    name: 'marketing / direct / email2',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e543',
    name: 'marketing / direct / email3',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e544',
    name: 'marketing / direct / email4',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e545',
    name: 'marketing / direct / email5',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e546',
    name: 'marketing / direct / email6',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e547',
    name: 'marketing / direct / email7',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e548',
    name: 'marketing / direct / email8',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e549',
    name: 'marketing / direct / email9',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e54a',
    name: 'marketing / direct / email10',
  },
  {
    id: '77369de3-2f4f-4c8f-86ce-952aef53e54b',
    name: 'marketing / direct / email11',
  },
];
describe('Tag', () => {
  it('should get a tag name from an id', () => {
    const tagID = '77369de3-2f4f-4c8f-86ce-952aef53e54b';
    const tagName = getTagNameFromID(systemTags, tagID);
    expect(tagName).toBe('marketing / direct / email11');
  });

  it('should return unknown tag when an id is not found', () => {
    const tagID = '12345678-2f4f-4c8f-86ce-952aef53e54b';
    const tagName = getTagNameFromID(systemTags, tagID);
    expect(tagName).toBe('unknown tag');
  });

  it('should filter tags', () => {
    const objectTags = ['77369de3-2f4f-4c8f-86ce-952aef53e54b'];
    const tags = getFilteredTags(objectTags, systemTags);
    expect(tags.length).toBe(systemTags.length - 1);
  });

  it('should not filter tags when there are no object tags', () => {
    const tags = getFilteredTags([], systemTags);
    expect(tags.length).toBe(systemTags.length);
  });

  it('should search filtered tags', () => {
    const search = 'email';
    const tags = getFilteredTagsAndSearch([], systemTags, search);
    expect(tags.length).toBe(12);
    tags.forEach((tag) => {
      expect(tag.name.includes('email'));
    });
  });

  it('should search filtered tags with no results', () => {
    const search = 'emailasdfsd';
    const tags = getFilteredTagsAndSearch([], systemTags, search);
    expect(tags.length).toBe(0);
  });
});
