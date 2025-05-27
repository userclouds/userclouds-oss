import { useState, useImperativeHandle, forwardRef, useRef } from 'react';
import TextInput, { TextInputProps } from '../TextInput';
import IconButton from '../IconButton';
import { IconEye, IconEyeOff } from '../../icons';

interface HiddenTextInputProps extends TextInputProps {
  hiddenInitially?: boolean;
}

const HiddenTextInput = forwardRef<HTMLInputElement, TextInputProps>(
  ({ hiddenInitially, ...otherProps }: HiddenTextInputProps, _forwardRef) => {
    const inputRef = useRef(null);
    useImperativeHandle(_forwardRef, () => inputRef.current);
    const [isHidden, setIsHidden] = useState<boolean>(
      typeof hiddenInitially !== 'undefined' ? hiddenInitially : true
    );
    const propsToPass: TextInputProps = { ...otherProps };
    delete propsToPass.type;

    return (
      <TextInput
        {...propsToPass}
        type={
          isHidden
            ? ('password' as TextInputProps['type'])
            : (otherProps as TextInputProps).type ||
              ('text' as TextInputProps['type'])
        }
        innerRight={
          !propsToPass.disabled && (
            <IconButton
              theme="clear"
              icon={isHidden ? <IconEye /> : <IconEyeOff />}
              title="Toggle text visibility"
              aria-label="Toggle text visibility"
              onClick={() => setIsHidden(!isHidden)}
            />
          )
        }
        ref={inputRef}
      />
    );
  }
);

HiddenTextInput.displayName = 'HiddenTextInput';

export default HiddenTextInput;
