export interface PageParameter {
  parameter_name: string;
  parameter_type: string;
  parameter_value: string;
}

export interface PageParametersResponse {
  client_id: string;
  page_parameters: PageParameter[];
  page_source_override: string;
}

// note:ksj: this is a small hack. we expect to change the API response
// format to obviate the need to do this munging
export const mungePageParameters = (
  pageParameters: PageParametersResponse
): Record<string, string> =>
  pageParameters.page_parameters.reduce(
    (acc: Record<string, string>, param: PageParameter) => {
      acc[param.parameter_name] = param.parameter_value;
      return acc;
    },
    { clientID: pageParameters.client_id }
  );
