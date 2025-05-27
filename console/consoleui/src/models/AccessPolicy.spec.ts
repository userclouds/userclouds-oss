import AccessPolicy, {
  AccessPolicyComponent,
  AccessPolicyType,
  blankAccessPolicyThresholds,
  blankPolicy,
  blankPolicyTemplate,
  ComponentPolicy,
  getIDForAccessPolicyComponent,
  getNameForAccessPolicyComponent,
  getNewComponentsOnNameChange,
  isTemplate,
} from './AccessPolicy';

const policy_component1: AccessPolicyComponent = {
  policy: {
    id: 'policy1',
    name: 'policy1',
  },
};

const policy_component2: AccessPolicyComponent = {
  policy: {
    id: 'policy2',
    name: 'policy2',
  },
};

const template_component1: AccessPolicyComponent = {
  template: {
    id: 'template1',
    name: 'template1',
    version: 0,
    description: '',
    function: '',
  },
  template_parameters: '',
};

const template_component2: AccessPolicyComponent = {
  template: {
    id: 'template2',
    name: 'template2',
    version: 0,
    description: '',
    function: '',
  },
  template_parameters: '',
};

const policy_component: ComponentPolicy = {
  isPolicy: true,
  policy: {
    id: 'policy',
    name: 'policy',
    version: 0,
    policy_type: AccessPolicyType.AND,
    tag_ids: [],
    components: [],
    required_context: {},
    thresholds: blankAccessPolicyThresholds(),
  },
  template: blankPolicyTemplate(),
  template_parameters: '',
};

const template_component: ComponentPolicy = {
  isPolicy: false,
  policy: blankPolicy(),
  template: {
    id: 'template',
    name: 'template',
    version: 0,
    description: '',
    function: '',
  },
  template_parameters: '',
};

describe('AccessPolicy', () => {
  it('should add a component policy to a policy', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [template_component1],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    newPolicy.components = getNewComponentsOnNameChange(
      'policy',
      [policy_component],
      newPolicy,
      0
    );

    expect(newPolicy.components.length).toBe(1);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy'
    );
  });

  it('should add a component template to a policy', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [policy_component1],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    newPolicy.components = getNewComponentsOnNameChange(
      'template',
      [template_component],
      newPolicy,
      0
    );

    expect(newPolicy.components.length).toBe(1);
    expect(isTemplate(newPolicy.components[0])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template'
    );
  });

  it('should change the correct template on a policy', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [policy_component1, template_component1],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );

    expect(isTemplate(newPolicy.components[1])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template1'
    );

    newPolicy.components = getNewComponentsOnNameChange(
      'template',
      [template_component],
      newPolicy,
      1
    );

    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(isTemplate(newPolicy.components[1])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template'
    );
  });

  it('should change the correct template on a policy', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [policy_component1, template_component1],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );

    expect(isTemplate(newPolicy.components[1])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template1'
    );

    newPolicy.components = getNewComponentsOnNameChange(
      'policy',
      [policy_component],
      newPolicy,
      1
    );

    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(isTemplate(newPolicy.components[1])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy'
    );
  });

  it('should change the correct policy on a policy by name', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [policy_component1, policy_component2],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );

    expect(isTemplate(newPolicy.components[1])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy2'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy2'
    );

    newPolicy.components = getNewComponentsOnNameChange(
      'policy',
      [policy_component],
      newPolicy,
      1
    );

    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'policy1'
    );
    expect(isTemplate(newPolicy.components[1])).toBeFalsy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'policy'
    );
  });

  it('should change the correct template on a policy by name', () => {
    const newPolicy: AccessPolicy = {
      id: '1',
      name: 'main',
      policy_type: AccessPolicyType.AND,
      tag_ids: [],
      version: 0,
      components: [template_component1, template_component2],
      required_context: {},
      thresholds: blankAccessPolicyThresholds(),
    };
    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template1'
    );

    expect(isTemplate(newPolicy.components[1])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template2'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template2'
    );

    newPolicy.components = getNewComponentsOnNameChange(
      'template',
      [template_component],
      newPolicy,
      1
    );

    expect(newPolicy.components.length).toBe(2);
    expect(isTemplate(newPolicy.components[0])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template1'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[0])).toBe(
      'template1'
    );

    expect(isTemplate(newPolicy.components[1])).toBeTruthy();
    expect(getIDForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template'
    );
    expect(getNameForAccessPolicyComponent(newPolicy.components[1])).toBe(
      'template'
    );
  });
});
