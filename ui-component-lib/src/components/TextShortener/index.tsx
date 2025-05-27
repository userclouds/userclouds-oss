import clsx from 'clsx';
import styles from './index.module.scss';

import { IconCopy } from '../../icons';
import IconButton from '../IconButton';
import Text from '../Text';

export interface TextShortenerProps {
  /** The text to display. */
  text?: string;
  /** Show lock icon for items that cannot be edited. */
  isCopyable?: boolean;
  /** How many characters should be displayed. Defaults to 8. */
  length?: number;
  /** Use sparingly */
  className?: string;
  /* Copy text (if different than text) */
  copyText?: string;
}

/** Shortens text to desired length. Adds copy button is applicable. */
export default function TextShortener({
  text = '',
  isCopyable = true,
  length = 8,
  className = '',
  copyText = undefined,
}: TextShortenerProps): JSX.Element {
  return (
    <div className={clsx(styles.root, className)} title={text}>
      {isCopyable && <CopyIcon copyText={copyText || text} />}
      {text && (
        <Text
          elementName="span"
          className={styles.text}
          style={{
            width: `${String(length)}ch`,
          }}
        >
          {text.length > length ? `${text.substring(0, length)}…` : text}
        </Text>
      )}
    </div>
  );
}

const CopyIcon = ({ copyText }: { copyText: string }) => {
  return (
    <>
      <IconButton
        icon={<IconCopy className={clsx(styles.copyIcon)} />}
        size="tiny"
        onClick={() => {
          navigator.clipboard.writeText(copyText);
        }}
        title="Copy to Clipboard"
        aria-label="Copy to Clipboard"
      />
      &nbsp;
    </>
  );
};
