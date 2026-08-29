import React, { useEffect, useMemo, useState } from 'react';
import { ArrowLeft, ArrowRight, Check, Compass, X } from 'lucide-react';
import { Button, IconButton } from '../../design-system';
import { availableTourSteps, nextTourIndex, previousTourIndex, productTourSteps } from './tour';
import { useTranslation } from 'react-i18next';

export const ProductTour: React.FC<{ open: boolean; onClose: () => void }> = ({ open, onClose }) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(0);
  const [rect, setRect] = useState<DOMRect | null>(null);
  const steps = useMemo(() => availableTourSteps(productTourSteps, (selector) => Boolean(document.querySelector(selector))), [open]);
  const step = steps[index];
  useEffect(() => { if (open) setIndex(0); }, [open]);
  useEffect(() => {
    if (!open || !step) return;
    const target = document.querySelector(step.target) as HTMLElement | null;
    if (!target) return;
    target.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    const update = () => setRect(target.getBoundingClientRect());
    update();
    window.addEventListener('resize', update);
    window.addEventListener('scroll', update, true);
    return () => { window.removeEventListener('resize', update); window.removeEventListener('scroll', update, true); };
  }, [open, step]);
  useEffect(() => { if (index >= steps.length) setIndex(Math.max(0, steps.length - 1)); }, [index, steps.length]);
  if (!open || !step || !rect) return null;
  const below = rect.bottom + 190 < window.innerHeight;
  const top = below ? Math.min(window.innerHeight - 190, rect.bottom + 12) : Math.max(12, rect.top - 178);
  const left = Math.min(Math.max(12, rect.left), Math.max(12, window.innerWidth - 360));
  return <div className="nx-tour-layer" role="dialog" aria-modal="true" aria-label={t('tour.aria')}>
    <div className="nx-tour-highlight" style={{ top: rect.top - 5, left: rect.left - 5, width: rect.width + 10, height: rect.height + 10 }} />
    <div className="nx-tour-card" style={{ top, left }}>
      <header><span><Compass size={15} /> {t('tour.label',{current:index+1,total:steps.length})}</span><IconButton label={t('tour.close')} onClick={onClose}><X size={14} /></IconButton></header>
      <strong>{t(step.title)}</strong><p>{t(step.body)}</p>
      <footer><Button size="sm" tone="ghost" disabled={index === 0} onClick={() => setIndex(previousTourIndex(index))}><ArrowLeft size={13} /> {t('tour.back')}</Button>{index === steps.length - 1 ? <Button size="sm" tone="brand" onClick={onClose}><Check size={13} /> {t('tour.finish')}</Button> : <Button size="sm" tone="brand" onClick={() => setIndex(nextTourIndex(index, steps.length))}>{t('tour.next')} <ArrowRight size={13} /></Button>}</footer>
    </div>
  </div>;
};
