export const state = {
  currentDir: '',
  files: [],
  fileMeta: {},
  currentIndex: -1,
  projectConfig: null,
  groupSelections: {},   // key -> Set (multi-select) or string|null (single-select)
  mruByGroup: {},        // key -> array of MRU values
  userSettings: {},
};
