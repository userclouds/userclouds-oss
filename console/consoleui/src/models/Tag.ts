export type TagModel = {
  id: string;
  name: string;
};

export const getTagNameFromID = (systemTags: TagModel[], tagID: string) => {
  const tag = systemTags.filter((t) => (t.id === tagID ? t.name : false))[0];
  return tag?.name || 'unknown tag';
};

export const getFilteredTags = (
  objectTags: string[],
  systemTags: TagModel[]
) => {
  const tags = systemTags.filter((tag) => !objectTags.includes(tag.id));
  return tags;
};

export const getFilteredTagsAndSearch = (
  objectTags: string[],
  systemTags: TagModel[],
  search: string
) => {
  const tags = getFilteredTags(objectTags, systemTags);
  return tags.filter((tag) => tag.name.includes(search));
};
