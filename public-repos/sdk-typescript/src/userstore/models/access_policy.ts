import ResourceID from './resource_id';

class AccessPolicyComponent {
  policy: ResourceID | null;

  template: ResourceID | null;

  template_parameters: string;

  constructor(
    policy: ResourceID | null,
    template: ResourceID | null,
    templateParameters: string
  ) {
    this.policy = policy;
    this.template = template;
    this.template_parameters = templateParameters;
  }

  static fromJSON(json: {
    [key: string]: string | object | null;
  }): AccessPolicyComponent {
    if (json.policy) {
      return new AccessPolicyComponent(
        ResourceID.fromJSON(json.policy as { [key: string]: string }),
        null,
        ''
      );
    }
    if (json.template) {
      return new AccessPolicyComponent(
        null,
        ResourceID.fromJSON(json.template as { [key: string]: string }),
        json.template_parameters as string
      );
    }
    return new AccessPolicyComponent(null, null, '');
  }

  toJSON() {
    if (this.policy)
      return {
        policy: this.policy,
      };
    if (this.template) {
      return {
        template: this.template,
        template_parameters: this.template_parameters,
      };
    }
    return {};
  }
}

type AccessPolicy = {
  id: string;
  name: string;
  description: string;
  policy_type: string;
  version: number;
  components: Array<AccessPolicyComponent>;
  required_context: Record<string, string>;
};

export default AccessPolicy;

export { AccessPolicyComponent };
