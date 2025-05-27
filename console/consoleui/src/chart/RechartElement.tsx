import { useState, useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  LegendProps,
} from 'recharts';
import { Payload as RechartMouseEnterProps } from 'recharts/types/component/DefaultLegendContent';

import { TabGroup } from '@userclouds/ui-component-lib';
import { QuerySet } from '../models/Chart';
import styles from './RechartElement.module.css';
import { chartLabels } from './ChartMetadataPerService';

const determineLabelAndPeriod = (timePeriod: string) => {
  let timeLabel;
  let axisLabel;

  switch (timePeriod) {
    case 'day':
      axisLabel = 'Hours';
      timeLabel = 'Hours Back';
      break;
    case 'week':
      axisLabel = 'Days';
      timeLabel = 'Days Back';
      break;
    default:
      axisLabel = 'Minutes';
      timeLabel = 'Minutes Back';
  }
  return { timeLabel, axisLabel };
};

const RechartElement = ({
  querySets,
  timePeriod,
}: {
  querySets: QuerySet[];
  timePeriod: string;
}) => {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const selectedChartData = querySets[selectedIndex].data;
  const { timeLabel, axisLabel } = determineLabelAndPeriod(timePeriod);

  // Generate the data keys from for Rechart from the selected querySet
  const dataKeys = useMemo(() => {
    if (
      !querySets ||
      !querySets[selectedIndex].data ||
      !querySets[selectedIndex]?.data?.[0]
    ) {
      return [];
    }

    const keys = Object.keys(querySets[selectedIndex]?.data?.[0] ?? {}).filter(
      (key) => key !== 'xAxis'
    );
    return keys;
  }, [querySets, selectedIndex]);

  // Set initial opacity of the items to 1
  const [opacity, setOpacity] = useState<Record<string, number>>(
    dataKeys.reduce<Record<string, number>>((acc, key) => {
      acc[key] = 1;
      return acc;
    }, {})
  );

  const handleMouseEnter = (o: RechartMouseEnterProps) => {
    const { dataKey } = o;
    // Set the non-hovered keys to 20% opacity
    setOpacity(() => {
      const opacities: Record<string, number> = {};
      dataKeys.forEach((key) => {
        opacities[key] = 0.2;
      });
      opacities[dataKey as string] = 1;
      return opacities;
    });
  };

  const handleMouseLeave = () => {
    // Reset all opacities to 1 when mouse leaves
    setOpacity(
      dataKeys.reduce<Record<string, number>>((acc, key) => {
        acc[key] = 1;
        return acc;
      }, {})
    );
  };

  if (!querySets || querySets.length === 0) {
    return null;
  }

  const handleTabChange = (tabId: string) => {
    const index = parseInt(tabId.split('-')[1], 10);
    setSelectedIndex(index);
  };

  const tabItems = querySets.map((querySet, i) => ({
    id: `querySet-${i}`,
    children: querySet.label,
  }));

  return (
    <div>
      {/* Only show toggle if there is more than one querySet */}
      {querySets.length > 1 && (
        <TabGroup
          items={tabItems}
          defaultActiveTab={`querySet-${selectedIndex}`}
          onTabChange={handleTabChange}
          className={styles.tabGroup}
        />
      )}

      <ResponsiveContainer
        width="100%"
        height="100%"
        minHeight={400}
        minWidth={400}
        debounce={100}
      >
        <AreaChart
          width={500}
          height={400}
          data={selectedChartData}
          margin={{
            top: 10,
            right: 30,
            left: 0,
            bottom: 0,
          }}
        >
          <defs>
            {dataKeys.map((key, index) => (
              <linearGradient
                key={`gradient-${key}`}
                id={`gradient-${key}`}
                x1="0"
                y1="0"
                x2="0"
                y2="1"
              >
                <stop
                  offset="20%"
                  stopColor={ColourPalette[index]}
                  stopOpacity={0.3}
                />
                <stop
                  offset="70%"
                  stopColor={ColourPalette[index]}
                  stopOpacity={0.05}
                />
              </linearGradient>
            ))}
          </defs>
          <CartesianGrid strokeDasharray="4 4" stroke="#f0f0f0" />
          <XAxis
            dataKey="xAxis"
            tickFormatter={(value) => {
              const numberValue = Number(value);
              return isNaN(numberValue) ? value : `${numberValue + 1}`; // Add 1 to show correct time period, and not 0
            }}
          />
          <YAxis />
          <Tooltip content={<CustomTooltip labelValue={timeLabel} />} />
          {dataKeys.map((key, index) => {
            return (
              <Area
                connectNulls
                key={key}
                type="monotone"
                dataKey={key}
                stackId={index}
                stroke={ColourPalette[index]}
                fill={`url(#gradient-${key})`}
                name={chartLabels.get(key) || key}
                opacity={opacity?.[key]}
                style={{
                  transition: 'opacity 0.3s ease',
                }}
              />
            );
          })}
          <Legend
            verticalAlign="top"
            wrapperStyle={{
              paddingBottom: '2rem',
            }}
            content={
              <ChartLegendContent
                handleMouseEnter={handleMouseEnter}
                handleMouseLeave={handleMouseLeave}
              />
            }
          />
        </AreaChart>
      </ResponsiveContainer>
      <div className={styles.timeSelection}>{axisLabel}</div>
    </div>
  );
};

export default RechartElement;

const ChartLegendContent = ({
  payload,
  handleMouseEnter,
  handleMouseLeave,
}: React.ComponentProps<'div'> &
  Pick<LegendProps, 'payload' | 'verticalAlign'> & {
    handleMouseEnter: (o: RechartMouseEnterProps) => void;
    handleMouseLeave: (o: RechartMouseEnterProps) => void;
  }) => {
  if (!payload?.length) {
    return null;
  }

  return (
    <div className={styles.legendContainer}>
      {payload.map((item) => {
        return (
          <div
            key={item.value}
            className={styles.legendItem}
            onMouseEnter={() => handleMouseEnter(item)}
            onMouseLeave={() => handleMouseLeave(item)}
          >
            <div
              className={styles.legendItemColor}
              style={{
                backgroundColor: item.color,
              }}
            />

            {item.value}
          </div>
        );
      })}
    </div>
  );
};

const CustomTooltip = ({
  active,
  payload,
  label,
  labelValue,
}: {
  active?: boolean;
  payload?: any;
  label?: string;
  labelValue?: string;
}): JSX.Element | null => {
  if (active && payload && payload.length) {
    const numberLabel = Number(label);
    const value = isNaN(numberLabel) ? label : `${numberLabel + 1}`; // Add 1 to show correct time period, and not 0

    return (
      <div className={styles.tooltip}>
        <i>
          {value} {labelValue}
        </i>
        {payload.map((p: any) => {
          const key = chartLabels.get(p.dataKey) || p.dataKey;
          return (
            <div
              style={{ color: p.color }}
              key={key}
              className={styles.tooltipItem}
            >
              <dt className={styles.tooltipItemKey}>{key}:</dt>
              <dd className={styles.tooltipItemValue}>{p.value}</dd>
            </div>
          );
        })}
      </div>
    );
  }

  return null;
};

const ColourPalette = [
  'hsl(292, 41.4%, 42.5%)',
  'hsl(188.7, 54.5%, 22.7%)',
  'hsl(47.9, 35.8%, 53.1%)',
  'hsl(217.2, 21.2%, 59.8%)',
  'hsl(160.1, 34.1%, 39.4%)',
  'hsl(60.5, 40.2%, 38.2%)',
  'hsl(351.3, 34.5%, 71.4%)',
];
