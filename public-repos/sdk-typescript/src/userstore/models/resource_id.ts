class ResourceID {
  id: string;

  name: string;

  constructor(id: string, name: string) {
    this.id = id;
    this.name = name;
  }

  static fromJSON(json: { [key: string]: string }): ResourceID {
    return new ResourceID(json.id, json.name);
  }

  toJSON() {
    if (this.id && this.name) {
      return {
        id: this.id,
        name: this.name,
      };
    }
    if (this.id) {
      return {
        id: this.id,
      };
    }
    if (this.name) {
      return {
        name: this.name,
      };
    }
    return {};
  }
}

export default ResourceID;
