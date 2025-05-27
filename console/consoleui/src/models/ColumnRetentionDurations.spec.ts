import {
  DurationUnit,
  DurationType,
  RetentionDuration,
  displayDuration,
} from './ColumnRetentionDurations';

let duration;
describe('ColumnRetentionDuration model', () => {
  describe('displayDuration', () => {
    it('should return "don\t auto-delete" if the duration type is Live and the unit is Indefinite', () => {
      duration = {
        unit: DurationUnit.Indefinite,
        duration: 0,
      } as RetentionDuration;
      expect(displayDuration(duration, DurationType.Live)).toBe(
        "don't auto-delete"
      );

      duration = {
        unit: 'indefinite',
        duration: 3,
      } as unknown as RetentionDuration;
      expect(displayDuration(duration, 'live' as DurationType)).toBe(
        "don't auto-delete"
      );

      duration = {
        unit: DurationUnit.Month,
        duration: 3,
      } as RetentionDuration;
      expect(displayDuration(duration, 'live' as DurationType)).not.toBe(
        "don't auto-delete"
      );

      duration = {
        unit: 'indefinite',
        duration: 3,
      } as unknown as RetentionDuration;
      expect(displayDuration(duration, DurationType.SoftDeleted)).not.toBe(
        "don't auto-delete"
      );
    });

    it('should return "don\t retain" if the duration type is SoftDeleted and the duration is 0', () => {
      duration = {
        unit: DurationUnit.Week,
        duration: 0,
      } as RetentionDuration;
      expect(displayDuration(duration, 'softdeleted' as DurationType)).toBe(
        "don't retain"
      );

      duration = {
        unit: 'minute',
        duration: 0,
      } as unknown as RetentionDuration;
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );

      duration = {
        unit: 'minute',
        duration: 1,
      } as unknown as RetentionDuration;
      expect(displayDuration(duration, DurationType.SoftDeleted)).not.toBe(
        "don't retain"
      );
    });

    it('should pluralize the unit if the quantity is 0', () => {
      duration = {
        unit: DurationUnit.Week,
        duration: 0,
      };
      expect(displayDuration(duration, DurationType.Live)).toBe('0 weeks');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );

      duration.unit = DurationUnit.Month;
      expect(displayDuration(duration, DurationType.Live)).toBe('0 months');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );

      duration.unit = DurationUnit.Year;
      expect(displayDuration(duration, DurationType.Live)).toBe('0 years');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );

      duration.unit = DurationUnit.Hour;
      expect(displayDuration(duration, DurationType.Live)).toBe('0 hours');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );

      duration.unit = DurationUnit.Day;
      expect(displayDuration(duration, DurationType.Live)).toBe('0 days');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        "don't retain"
      );
    });

    it('should pluralize the unit if the quantity is greater than 1', () => {
      duration = {
        unit: DurationUnit.Week,
        duration: 2,
      };
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '2 weeks'
      );
      expect(displayDuration(duration, DurationType.Live)).toBe('2 weeks');

      duration = {
        unit: DurationUnit.Month,
        duration: 6,
      };
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '6 months'
      );
      expect(displayDuration(duration, DurationType.Live)).toBe('6 months');

      duration = {
        unit: DurationUnit.Year,
        duration: 38,
      };
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '38 years'
      );
      expect(displayDuration(duration, DurationType.Live)).toBe('38 years');

      duration = {
        unit: DurationUnit.Hour,
        duration: 162,
      };
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '162 hours'
      );
      expect(displayDuration(duration, DurationType.Live)).toBe('162 hours');

      duration = {
        unit: DurationUnit.Day,
        duration: 3,
      };
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '3 days'
      );
      expect(displayDuration(duration, DurationType.Live)).toBe('3 days');
    });

    it('should not pluralize the unit if the quantity is 1', () => {
      duration = {
        unit: DurationUnit.Week,
        duration: 1,
      };
      expect(displayDuration(duration, DurationType.Live)).toBe('1 week');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '1 week'
      );

      duration.unit = DurationUnit.Month;
      expect(displayDuration(duration, DurationType.Live)).toBe('1 month');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '1 month'
      );

      duration.unit = DurationUnit.Year;
      expect(displayDuration(duration, DurationType.Live)).toBe('1 year');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '1 year'
      );

      duration.unit = DurationUnit.Hour;
      expect(displayDuration(duration, DurationType.Live)).toBe('1 hour');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe(
        '1 hour'
      );

      duration.unit = DurationUnit.Day;
      expect(displayDuration(duration, DurationType.Live)).toBe('1 day');
      expect(displayDuration(duration, DurationType.SoftDeleted)).toBe('1 day');
    });
  });
});
