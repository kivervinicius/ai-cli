import React from 'react';
import { useTranslation } from 'react-i18next';
import { ShieldCheck } from 'lucide-react';
import { Badge, Card } from '../../design-system';
import type { AutonomyContract } from '../../types';
import { linesToList, listToLines } from './missionAutonomyModel';
import styles from './MissionAutonomyCard.module.scss';

export const MissionAutonomyCard: React.FC<{
  value: AutonomyContract;
  onChange: (value: AutonomyContract) => void;
}> = ({ value, onChange }) => {
  const { t } = useTranslation();
  const patch = <K extends keyof AutonomyContract>(key: K, next: AutonomyContract[K]) => {
    onChange({ ...value, [key]: next });
  };

  const toggle = (
    key: keyof Pick<
      AutonomyContract,
      | 'auto_remediate'
      | 'require_verification'
      | 'disallow_destructive_git'
      | 'escalate_on_failure'
      | 'allow_tool_auto_approval'
      | 'allow_git_push'
      | 'allow_deploy'
      | 'allow_external_network'
      | 'allow_secret_access'
      | 'allow_paid_services'
    >,
    label: string,
    dangerous = false,
  ) => (
    <label className={styles.toggleLabel}>
      <input
        type="checkbox"
        checked={value[key]}
        onChange={(event) => patch(key, event.target.checked)}
      />
      <span>{label}</span>
      {dangerous && value[key] ? (
        <Badge tone="warning">{t('autonomyContract.explicitlyAllowed')}</Badge>
      ) : null}
    </label>
  );

  const sensitiveEnabled = [
    value.allow_git_push,
    value.allow_deploy,
    value.allow_external_network,
    value.allow_secret_access,
    value.allow_paid_services,
  ].filter(Boolean).length;

  const numberField = (
    key: 'max_retries' | 'max_total_iterations' | 'max_no_progress' | 'package_timeout_seconds',
    label: string,
    min: number,
  ) => (
    <label className={styles.fieldLabel}>
      <span>{label}</span>
      <input
        type="number"
        min={min}
        value={value[key]}
        onChange={(event) => patch(key, Math.max(min, Number(event.target.value) || min))}
        className={styles.numberInput}
      />
    </label>
  );

  return (
    <Card className={styles.card}>
      <div className={styles.header}>
        <div className={styles.title}>
          <ShieldCheck size={15} /> {t('autonomyContract.title')}
        </div>
        <Badge tone={sensitiveEnabled > 0 ? 'warning' : 'success'}>
          {sensitiveEnabled > 0
            ? t('autonomyContract.sensitiveGrants', { count: sensitiveEnabled })
            : t('autonomyContract.localFirst')}
        </Badge>
      </div>
      <p className={styles.intro}>{t('autonomyContract.intro')}</p>

      <div className={styles.numbersGrid}>
        {numberField('max_retries', t('autonomyContract.retries'), 1)}
        {numberField('max_total_iterations', t('autonomyContract.iterationBudget'), 1)}
        {numberField('max_no_progress', t('autonomyContract.noProgressLimit'), 1)}
        {numberField('package_timeout_seconds', t('autonomyContract.packageTimeout'), 30)}
      </div>

      <div className={styles.togglesList}>
        {toggle('require_verification', t('autonomyContract.requireVerification'))}
        {toggle('auto_remediate', t('autonomyContract.autoRemediate'))}
        {toggle('escalate_on_failure', t('autonomyContract.escalateOnFailure'))}
        {toggle('disallow_destructive_git', t('autonomyContract.disallowDestructiveGit'))}
        {toggle('allow_tool_auto_approval', t('autonomyContract.allowToolAutoApproval'))}
      </div>

      <details className={styles.details}>
        <summary className={styles.detailsSummary}>{t('autonomyContract.sensitivePerms')}</summary>
        <div className={styles.detailsContent}>
          {toggle('allow_git_push', t('autonomyContract.allowGitPush'), true)}
          {toggle('allow_deploy', t('autonomyContract.allowDeploy'), true)}
          {toggle('allow_external_network', t('autonomyContract.allowExternalNetwork'), true)}
          {toggle('allow_secret_access', t('autonomyContract.allowSecretAccess'), true)}
          {toggle('allow_paid_services', t('autonomyContract.allowPaidServices'), true)}
        </div>
      </details>

      <label className={styles.textareaField}>
        <span>{t('autonomyContract.allowedFilePatterns')}</span>
        <textarea
          rows={3}
          value={listToLines(value.allowed_file_patterns)}
          onChange={(event) => patch('allowed_file_patterns', linesToList(event.target.value))}
          placeholder={'src/**\ntests/**'}
          className={styles.textarea}
        />
      </label>

      <label className={styles.textareaField}>
        <span>{t('autonomyContract.verificationCommands')}</span>
        <textarea
          rows={3}
          value={listToLines(value.verification_commands)}
          onChange={(event) => patch('verification_commands', linesToList(event.target.value))}
          placeholder={'go test ./...\nnpm test'}
          disabled={!value.require_verification}
          className={styles.textarea}
        />
      </label>
    </Card>
  );
};
