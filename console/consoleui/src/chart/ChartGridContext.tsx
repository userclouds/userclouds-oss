import React from 'react';

type ChartGridContextType = {
  service: string;
  timePeriod: string;
  endTime: string;
};

const defaultContext: ChartGridContextType = {
  service: 'plex',
  timePeriod: 'day',
  endTime: '',
};

const ChartGridContext =
  React.createContext<ChartGridContextType>(defaultContext);

export default ChartGridContext;
export type { ChartGridContextType };
