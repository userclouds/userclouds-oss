import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
  tryGetJSON,
} from '@userclouds/sharedui';

import { makeCompanyConfigURL } from '../API';
import { countQueryToJSON, type CountQuery } from '../models/CountQuery';
import { CountMetric } from '../models/Metrics';
import ChartQuery from '../chart/ChartQuery';
import { ChartRenderableData, ChartResponse } from '../models/Chart';

export const fetchCountData = async (
  tenantID: string,
  query: CountQuery
): Promise<CountMetric[]> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/counters/query`
    );

    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(countQueryToJSON(query)),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(error, `error fetching data (${JSON.stringify(query)})`)
        );
      });
  });
};

export const fetchChartData = async (
  tenantID: string,
  queries: ChartQuery[]
): Promise<ChartRenderableData[][]> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/counters/charts`
    );

    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(queries),
      });
      const jsonResponse = (await tryGetJSON(rawResponse)) as ChartResponse;

      if (jsonResponse?.charts) {
        // Clean up Chart data: flatten nested structure from the API response.
        const flattenedCharts = jsonResponse.charts.map(({ chart }) => {
          return chart.map(({ column }) => {
            return {
              xAxis: column[0].xAxis,
              ...column[0].values,
            } as ChartRenderableData;
          });
        });

        resolve(flattenedCharts);
      } else {
        reject(
          makeAPIError(
            new Error('Invalid JSON response'),
            `error fetching data (${JSON.stringify(queries)})`
          )
        );
      }
    } catch (e) {
      reject(
        makeAPIError(e, `error fetching data (${JSON.stringify(queries)})`)
      );
    }
  });
};
