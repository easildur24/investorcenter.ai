declare module 'react-gauge-chart' {
  import { FC } from 'react';

  export interface GaugeChartProps {
    id: string;
    nrOfLevels?: number;
    colors?: string[];
    arcWidth?: number;
    percent: number;
    textColor?: string;
    needleColor?: string;
    needleBaseColor?: string;
    hideText?: boolean;
    formatTextValue?: (value: string) => string;
    animDelay?: number;
    animateDuration?: number;
  }

  const GaugeChart: FC<GaugeChartProps>;
  export default GaugeChart;
}
