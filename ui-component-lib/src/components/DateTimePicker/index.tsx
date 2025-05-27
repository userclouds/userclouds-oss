import clsx from 'clsx';
// eslint-disable-next-line import/no-named-default
import { default as Picker } from 'react-datetime-picker';
import styles from './index.module.scss';
import TextInputStyles from '../TextInput/index.module.scss';

/**
 * This is built on https://www.npmjs.com/package/react-datetime-picker,
 * which uses https://www.npmjs.com/package/react-calendar.
 * All of the props for either of those components can be passed in to DateTimePicker.
 * Make sure to pass a Date object as a value and suppy an onChange function.
 * to the `id` of the input.
 */
const DateTimePicker = (props: React.ComponentProps<typeof DateTimePicker>) => {
  const { value, onChange, ...otherProps } = props;
  return (
    <Picker
      value={value}
      disableClock
      format="MM/dd/yyyy h:mm:ss a"
      calendarType="US"
      onChange={onChange}
      className={clsx(TextInputStyles.medium, styles['react-datetime-picker'])}
      {...otherProps}
    />
  );
};

export default DateTimePicker;
