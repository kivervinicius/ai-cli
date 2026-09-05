import React from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import './i18n';
import './styles/globals.scss';

const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(<App />);
}
