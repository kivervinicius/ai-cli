import React from 'react';
import { Spinner } from '../design-system';
import styles from './SurfaceLoadingFallback.module.scss';

export const SurfaceLoadingFallback: React.FC<{ label?: string }> = ({
  label = 'Carregando superfície…',
}) => {
  return (
    <div className={styles.container} role="status" aria-live="polite">
      <Spinner label={label} />
      <span className={styles.text}>{label}</span>
    </div>
  );
};
