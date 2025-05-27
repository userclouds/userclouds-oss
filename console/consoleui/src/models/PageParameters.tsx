interface PageParameterValue {
  name: string;
  current_value: string;
  default_value: string;
}
export type PageParametersForPage = Record<string, PageParameterValue>;

interface PageParameterSaveValue {
  parameter_name: string;
  new_value: string;
}
type PageParametersForSave = Record<string, PageParameterSaveValue>;

export interface PageParametersResponse {
  tenant_id: string;
  app_id: string;
  page_type_parameters: Record<string, PageParametersForPage>;
}

export interface PageParametersSavePayload {
  page_type_parameter_changes: Record<string, PageParametersForSave>;
}

type ImageFormat = 'gif' | 'jpeg' | 'png';

export interface ImageUploadResponse {
  tenant_id: string;
  app_id: string;
  image_url: string;
  image_meta_data: {
    format: ImageFormat;
    height: number;
    size: number;
    type: string;
    width: number;
  };
}

export const updatePageParameters = (
  params: PageParametersResponse,
  pageName: string,
  paramName: string,
  value: string
): PageParametersResponse => {
  return {
    ...params,
    page_type_parameters: {
      ...params.page_type_parameters,
      [pageName]: {
        ...params.page_type_parameters[pageName],
        [paramName]: {
          ...params.page_type_parameters[pageName][paramName],
          current_value: value,
        },
      },
    },
  };
};

export const pageParametersForPreview = (
  params: PageParametersResponse,
  pageName: string
) => {
  return {
    ...Object.keys(params.page_type_parameters.every_page).reduce(
      (acc: Record<string, string>, k: string) => {
        acc[k] = params.page_type_parameters.every_page[k].current_value;
        return acc;
      },
      {}
    ),
    ...Object.keys(params.page_type_parameters[pageName]).reduce(
      (acc: Record<string, string>, k: string) => {
        acc[k] = params.page_type_parameters[pageName][k].current_value;
        return acc;
      },
      {}
    ),
  };
};

export const pageParametersForSave = (
  params: PageParametersResponse
): PageParametersSavePayload => {
  return {
    page_type_parameter_changes: Object.keys(
      params.page_type_parameters
    ).reduce(
      (
        acc: Record<string, PageParametersForSave>,
        pageName: string
      ): Record<string, PageParametersForSave> => {
        const paramsForPage = Object.keys(
          params.page_type_parameters[pageName]
        ).reduce((pageParams: PageParametersForSave, paramName: string) => {
          pageParams[paramName] = {
            parameter_name: paramName,
            new_value:
              params.page_type_parameters[pageName][paramName].current_value,
          } as PageParameterSaveValue;
          return pageParams;
        }, {});
        acc[pageName] = paramsForPage;
        return acc;
      },
      {}
    ),
  };
};

export const toggleArrayParam = (
  valueString: string,
  valueToToggle: string,
  add: boolean
) => {
  const asSet =
    valueString.length === 0 ? new Set() : new Set(valueString.split(','));
  const method = add ? 'add' : 'delete';
  asSet[method](valueToToggle);
  return [...asSet].join(',');
};

export const arrayParamAsSet = (
  params: PageParametersResponse,
  pageName: string,
  paramName: string
) => {
  const paramValue =
    params.page_type_parameters[pageName][paramName].current_value;
  return paramValue.length === 0 ? new Set() : new Set(paramValue.split(','));
};
