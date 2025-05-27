import { connect } from 'react-redux';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState } from '../store';
import { redirect } from '../routing';
import { createAccessPolicyTemplate } from '../thunks/tokenizer';
import {
  AccessPolicyTemplate,
  blankPolicyTemplate,
} from '../models/AccessPolicy';
import PolicyTemplateForm from './PolicyTemplateForm';

const cancel = (queryString: string) => () => {
  redirect(`/policytemplates${queryString}`);
};

const CreatePolicyTemplatePage = ({
  newTemplate,
  query,
}: {
  newTemplate: AccessPolicyTemplate | undefined;
  query: URLSearchParams;
}) => {
  const cleanQuery = makeCleanPageLink(query);

  return (
    <PolicyTemplateForm
      editableTemplate={newTemplate || blankPolicyTemplate()}
      savedTemplate={undefined}
      saveTemplate={createAccessPolicyTemplate(cleanQuery)}
      onCancel={cancel(cleanQuery)}
      isDialog={false}
    />
  );
};

export default connect((state: RootState) => ({
  newTemplate: state.policyTemplateToCreate,
  query: state.query,
}))(CreatePolicyTemplatePage);
