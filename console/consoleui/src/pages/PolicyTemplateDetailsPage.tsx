import { useEffect } from 'react';
import { connect } from 'react-redux';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';

import { AccessPolicyTemplate } from '../models/AccessPolicy';
import PolicyTemplateForm from './PolicyTemplateForm';
import { fetchPolicyTemplate, savePolicyTemplate } from '../thunks/tokenizer';

const PolicyTemplateDetailsPage = ({
  selectedTenantID,
  selectedTemplate,
  templateToModify,
  query,
  routeParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  selectedTemplate: AccessPolicyTemplate | undefined;
  templateToModify: AccessPolicyTemplate | undefined;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { templateID, version } = routeParams;
  const cleanQuery = makeCleanPageLink(query);

  useEffect(() => {
    if (selectedTenantID && templateID) {
      // TODO: find in redux
      dispatch(
        fetchPolicyTemplate(
          selectedTenantID,
          templateID,
          version && version !== 'latest' ? version : ''
        )
      );
    }
  }, [selectedTenantID, templateID, version, query, dispatch]);

  return (
    <PolicyTemplateForm
      editableTemplate={templateToModify}
      savedTemplate={selectedTemplate}
      saveTemplate={savePolicyTemplate(cleanQuery)}
      onCancel={() => undefined}
      isDialog={false}
    />
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  selectedTemplate: state.selectedPolicyTemplate,
  templateToModify: state.policyTemplateToModify,
  query: state.query,
  routeParams: state.routeParams,
}))(PolicyTemplateDetailsPage);
