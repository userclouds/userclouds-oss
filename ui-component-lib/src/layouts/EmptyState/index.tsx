import React from 'react';
import Heading from '../../components/Heading';
import Text from '../../components/Text';
import styles from './index.module.scss';

export interface EmptyStateProps {
  /** Content below the title, image, and subtitle. Often a call to action */
  children: React.ReactNode;
  /** Image to use above title */
  image?: React.ReactNode;
  /** Title of empty state */
  title?: string;
  /** Further description */
  subTitle?: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({
  children,
  title,
  subTitle,
  image,
}) => (
  <div className={styles.root}>
    <div className={styles.imageWrap}>{image}</div>
    <Heading className={styles.title} headingLevel={2} size={2}>
      {title}
    </Heading>
    {subTitle && (
      <Text className={styles.subTitle} size={1}>
        {subTitle}
      </Text>
    )}
    <div className={styles.buttonWrap}>{children}</div>
  </div>
);

export default EmptyState;
