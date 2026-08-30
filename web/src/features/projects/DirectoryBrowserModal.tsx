import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Folder,
  FolderGit2,
  Home,
  Monitor,
  FileText,
  HardDrive,
  CornerLeftUp,
  Search,
  Check,
  GitBranch,
  Loader2,
  ChevronRight,
  FolderPlus,
} from 'lucide-react';
import { Dialog, Button, Input, IconButton, Badge } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { FSBrowseResult, FSEntry } from '../../types';

export const DirectoryBrowserModal: React.FC<{
  open: boolean;
  onClose: () => void;
  initialPath?: string;
  onSelectPath: (path: string, suggestedName?: string) => void;
}> = ({ open, onClose, initialPath, onSelectPath }) => {
  const { t } = useTranslation();
  const [currentPath, setCurrentPath] = useState(initialPath || '');
  const [data, setData] = useState<FSBrowseResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [mkdirBusy, setMkdirBusy] = useState(false);
  const [mkdirError, setMkdirError] = useState('');

  const loadDirectory = async (path?: string) => {
    setLoading(true);
    setError('');
    try {
      const res = await nexus.browseFS(path);
      setData(res);
      setCurrentPath(res.current_path);
      setQuery('');
      setNewFolderOpen(false);
    } catch (err) {
      console.error('Failed to browse directory', err);
      // If a specific path was requested and failed, try default browse fallback (CWD/home)
      if (path) {
        try {
          const fallback = await nexus.browseFS();
          setData(fallback);
          setCurrentPath(fallback.current_path);
          setQuery('');
          setNewFolderOpen(false);
          return;
        } catch (e2) {
          setError(e2 instanceof Error ? e2.message : String(e2));
        }
      } else {
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      void loadDirectory(initialPath);
    }
  }, [open, initialPath]);

  const handleSelectCurrent = () => {
    if (!data) return;
    const name = data.current_path.split('/').filter(Boolean).pop() || 'Project';
    onSelectPath(data.current_path, name);
    onClose();
  };

  const handleSelectEntry = (entry: FSEntry) => {
    if (entry.is_dir) {
      void loadDirectory(entry.path);
    }
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim() || !data) return;
    setMkdirBusy(true);
    setMkdirError('');
    try {
      const target = `${data.current_path}/${newFolderName.trim()}`;
      await nexus.mkdirFS(target);
      setNewFolderName('');
      setNewFolderOpen(false);
      void loadDirectory(target);
    } catch (err) {
      setMkdirError(err instanceof Error ? err.message : String(err));
    } finally {
      setMkdirBusy(false);
    }
  };

  const filteredEntries = (data?.entries || []).filter(
    (e) => e.is_dir && e.name.toLowerCase().includes(query.toLowerCase())
  );

  const getBookmarkIcon = (icon: string) => {
    switch (icon) {
      case 'home':
        return <Home size={14} />;
      case 'desktop':
        return <Monitor size={14} />;
      case 'documents':
        return <FileText size={14} />;
      case 'root':
        return <HardDrive size={14} />;
      default:
        return <Folder size={14} />;
    }
  };

  const bookmarks = (data?.bookmarks && data.bookmarks.length > 0)
    ? data.bookmarks
    : [
        { label: 'Home', path: '~', icon: 'home' },
        { label: 'Projetos', path: '/projetos', icon: 'folder' },
        { label: 'Root', path: '/', icon: 'root' },
      ];

  const breadcrumbs = (data?.breadcrumbs && data.breadcrumbs.length > 0)
    ? data.breadcrumbs
    : (currentPath ? currentPath.split('/').filter(Boolean).reduce((acc: string[], curr: string) => [...acc, `${acc[acc.length - 1] || ''}/${curr}`], ['/']) : ['/']);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('projectManager.browseOS')}
      wide
    >
      <div className="nx-dir-picker">
        {/* Top OS Breadcrumbs bar */}
        <div className="nx-dir-picker__breadcrumbs">
          {data?.parent_path && (
            <IconButton
              label={t('projectManager.upDirectory')}
              onClick={() => loadDirectory(data.parent_path)}
            >
              <CornerLeftUp size={14} />
            </IconButton>
          )}

          <div className="nx-dir-breadcrumbs-list">
            {breadcrumbs.map((crumb, idx) => {
              const label = crumb === '/' ? '/' : crumb.split('/').filter(Boolean).pop();
              const isLast = idx === breadcrumbs.length - 1;
              return (
                <React.Fragment key={crumb}>
                  {idx > 0 && <ChevronRight size={12} className="nx-crumb-sep" />}
                  <button
                    type="button"
                    className={`nx-crumb-btn ${isLast ? 'nx-crumb-btn--active' : ''}`}
                    onClick={() => !isLast && loadDirectory(crumb)}
                    disabled={isLast}
                  >
                    {label}
                  </button>
                </React.Fragment>
              );
            })}
          </div>

          <div className="nx-dir-picker__top-actions">
            <Button
              size="sm"
              tone="ghost"
              onClick={() => setNewFolderOpen((prev) => !prev)}
            >
              <FolderPlus size={13} />
              <span>{t('projectManager.newFolder')}</span>
            </Button>
          </div>
        </div>

        {/* New Folder Inline Form */}
        {newFolderOpen && (
          <div className="nx-dir-mkdir-box">
            <Input
              value={newFolderName}
              onChange={setNewFolderName}
              placeholder={t('projectManager.folderName')}
              onEnter={handleCreateFolder}
              autoFocus
            />
            {mkdirError && <span className="nx-error-copy">{mkdirError}</span>}
            <div className="nx-mkdir-actions">
              <Button size="sm" onClick={() => setNewFolderOpen(false)}>
                {t('common.closeDialog')}
              </Button>
              <Button
                size="sm"
                tone="brand"
                disabled={!newFolderName.trim() || mkdirBusy}
                onClick={handleCreateFolder}
              >
                {t('projectManager.createFolder')}
              </Button>
            </div>
          </div>
        )}

        <div className="nx-dir-picker__body">
          {/* OS Quick Bookmarks Sidebar */}
          <div className="nx-dir-picker__sidebar">
            <span className="nx-dir-sidebar-heading">{t('projectManager.bookmarks')}</span>
            <div className="nx-dir-bookmarks-list">
              {bookmarks.map((b) => (
                <button
                  type="button"
                  key={b.path}
                  className={`nx-dir-bookmark-btn ${currentPath === b.path ? 'nx-dir-bookmark-btn--active' : ''}`}
                  onClick={() => loadDirectory(b.path)}
                >
                  {getBookmarkIcon(b.icon)}
                  <span>{b.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Directory Explorer Pane */}
          <div className="nx-dir-picker__main">
            <div className="nx-dir-search-bar">
              <Search size={13} className="nx-dir-search-icon" />
              <Input
                value={query}
                onChange={setQuery}
                placeholder={t('common.search')}
              />
            </div>

            {loading ? (
              <div className="nx-dir-loading">
                <Loader2 size={24} className="nx-spin" />
                <span>{t('common.loading')}</span>
              </div>
            ) : error ? (
              <div className="nx-dir-empty nx-dir-error">
                <p style={{ color: 'var(--nx-danger, #ef4444)', marginBottom: '12px' }}>{error}</p>
                <Button size="sm" tone="brand" onClick={() => loadDirectory(currentPath || undefined)}>
                  <CornerLeftUp size={13} />
                  <span>{t('directoryBrowser.retry')}</span>
                </Button>
              </div>
            ) : (
              <div className="nx-dir-entries-grid">
                {filteredEntries.map((entry) => (
                  <div
                    key={entry.path}
                    className="nx-dir-entry-card"
                    onClick={() => handleSelectEntry(entry)}
                  >
                    <div className="nx-dir-entry-icon">
                      {entry.is_git ? (
                        <FolderGit2 size={22} className="nx-git-folder-icon" />
                      ) : (
                        <Folder size={22} />
                      )}
                    </div>
                    <div className="nx-dir-entry-meta">
                      <strong>{entry.name}</strong>
                      <div className="nx-dir-entry-tags">
                        {entry.is_git && (
                          <span className="nx-git-tag">
                            <GitBranch size={10} /> Git
                          </span>
                        )}
                        {(entry.tech ?? []).slice(0, 2).map((t) => (
                          <span key={t} className="nx-tech-tag">
                            {t}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                ))}

                {filteredEntries.length === 0 && (
                  <div className="nx-dir-empty">
                    <p>{t('projectManager.noProjects')}</p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Footer with Selected Folder info & confirmation button */}
        <div className="nx-dir-picker__footer">
          <div className="nx-dir-selected-info">
            <span className="nx-dir-selected-label">{t('directoryBrowser.currentPath')}</span>
            <code>{currentPath || data?.current_path || '/'}</code>
            {data?.is_git && (
              <Badge tone="brand">
                <GitBranch size={11} /> {data.git_branch || 'git'}
              </Badge>
            )}
            {data?.tech && data.tech.length > 0 && (
              <Badge tone="default">{data.tech.join(', ')}</Badge>
            )}
          </div>

          <div className="nx-dir-footer-actions">
            <Button onClick={onClose}>{t('common.closeDialog')}</Button>
            <Button tone="brand" onClick={handleSelectCurrent}>
              <Check size={14} />
              <span>{t('projectManager.selectFolder')}</span>
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  );
};
