// Request types
interface RequestQuerySet {
  name: string;
  eventTypes: number[];
}

export interface RequestChartDetails {
  title: string;
  divId: string;
  querySets?: RequestQuerySet[];
}

export type RequestChartsMetadata = {
  [key: string]: {
    service: string;
    charts: RequestChartDetails[];
  };
};

// Response types
type ResponseChartData = {
  xAxis: string;
  values: Record<string, number>;
};

type ResponseChartRow = {
  column: ResponseChartData[];
};

export type ResponseChart = {
  chart: ResponseChartRow[];
};

export type ChartResponse = {
  charts: ResponseChart[];
};

// Renderable types
// This can have any number of data keys, and one col key
export type ChartRenderableData = {
  xAxis: string;
  [key: string]: number | string;
};

export type QuerySet = {
  data: ChartRenderableData[] | undefined;
  label: string;
};

export type QuerySets = {
  querySets: QuerySet[];
};
