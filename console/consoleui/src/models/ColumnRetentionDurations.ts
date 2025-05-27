export enum DurationUnit {
  Indefinite = 'indefinite',
  Year = 'year',
  Month = 'month',
  Week = 'week',
  Day = 'day',
  Hour = 'hour',
}

export enum DurationType {
  Live = 'live',
  SoftDeleted = 'softdeleted',
}

export type RetentionDuration = {
  unit: DurationUnit;
  duration: number;
};

export type PurposeRetentionDuration = {
  id: string;
  version: number;
  purpose_id: string;
  purpose_name: string;
  duration: RetentionDuration;
  use_default: boolean;
  default_duration: RetentionDuration;
};

export type ColumnRetentionDurationsResponse = {
  column_id: string;
  duration_type: string;
  max_duration: RetentionDuration;
  supported_duration_units: DurationUnit[];
  purpose_retention_durations: PurposeRetentionDuration[];
};

export const displayDuration = (
  duration: RetentionDuration,
  type: DurationType
) => {
  if (type === DurationType.Live && duration.unit === DurationUnit.Indefinite) {
    return "don't auto-delete";
  }
  if (type === DurationType.SoftDeleted && duration.duration === 0) {
    return "don't retain";
  }
  return `${duration.duration} ${duration.unit}${
    duration.duration === 1 ? '' : 's'
  }`;
};
