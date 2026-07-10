import { useEffect, useMemo, useState } from 'react';
import './style.css';

const options = {
  frontend: ['react', 'vue', 'none'],
  backend: ['go', 'node', 'python', 'php', 'java', 'none'],
  database: ['postgres', 'mysql', 'none'],
  cache: ['redis', 'none'],
};

const defaultConfig = {
  frontend: 'react',
  backend: 'go',
  database: 'postgres',
  cache: 'redis',
  nginx: true,
  localstack: false,
  monitoring: true,
  cicd: true,
  project: 'studio-sample',
};

function App() {
  const [config, setConfig] = useState(defaultConfig);
  const [preview, setPreview] = useState(null);
  const [selectedPath, setSelectedPath] = useState('');
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const selectedFile = useMemo(() => {
    if (!preview?.files?.length) return null;
    return preview.files.find((file) => file.path === selectedPath) || preview.files[0];
  }, [preview, selectedPath]);

  const groupedFiles = useMemo(() => {
    const groups = {};
    for (const file of preview?.files || []) {
      groups[file.kind] = groups[file.kind] || [];
      groups[file.kind].push(file);
    }
    return groups;
  }, [preview]);

  useEffect(() => {
    loadPreview();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function requestJSON(path, body) {
    const response = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Request failed with ${response.status}`);
    }
    return response.json();
  }

  async function loadPreview() {
    setLoading(true);
    setError('');
    setMessage('');
    try {
      const data = await requestJSON('/api/preview', config);
      setPreview(data);
      setSelectedPath((current) => {
        if (data.files.some((file) => file.path === current)) return current;
        return data.files[0]?.path || '';
      });
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  async function generateProject() {
    setGenerating(true);
    setError('');
    setMessage('');
    try {
      const data = await requestJSON('/api/generate', config);
      setMessage(`${data.message}. ${data.fileCount} files written.`);
    } catch (err) {
      setError(err.message);
    } finally {
      setGenerating(false);
    }
  }

  function updateField(field, value) {
    setConfig((current) => ({ ...current, [field]: value }));
  }

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Built with dev-box</p>
          <h1>DevBox Studio</h1>
          <p className="hero-copy">
            Choose a stack, preview the files that dev-box will create, then generate
            a ready-to-run project from the browser.
          </p>
        </div>
        <div className="hero-stats" aria-label="Current preview summary">
          <span>{preview?.files?.length || 0}</span>
          <small>generated files</small>
        </div>
      </section>

      <section className="workspace">
        <aside className="panel controls" aria-label="Stack controls">
          <div className="panel-heading">
            <h2>Stack</h2>
            <p>Every preview is produced by the local dev-box CLI.</p>
          </div>

          <div className="control-grid">
            {Object.entries(options).map(([field, values]) => (
              <label className="field" key={field}>
                <span>{labelFor(field)}</span>
                <select value={config[field]} onChange={(event) => updateField(field, event.target.value)}>
                  {values.map((value) => (
                    <option key={value} value={value}>
                      {display(value)}
                    </option>
                  ))}
                </select>
              </label>
            ))}
          </div>

          <div className="addons" aria-label="Add-ons">
            {['nginx', 'localstack', 'monitoring', 'cicd'].map((field) => (
              <label className="toggle" key={field}>
                <input
                  type="checkbox"
                  checked={config[field]}
                  onChange={(event) => updateField(field, event.target.checked)}
                />
                <span>{addonLabel(field)}</span>
              </label>
            ))}
          </div>

          <label className="field">
            <span>Output project</span>
            <input
              value={config.project}
              onChange={(event) => updateField('project', event.target.value)}
              placeholder="studio-sample"
            />
          </label>

          <div className="actions">
            <button className="primary" onClick={loadPreview} disabled={loading}>
              {loading ? 'Previewing...' : 'Preview files'}
            </button>
            <button className="secondary" onClick={generateProject} disabled={generating || loading}>
              {generating ? 'Generating...' : 'Generate project'}
            </button>
          </div>

          {preview?.summary && <p className="status good">{preview.summary}</p>}
          {message && <p className="status good">{message}</p>}
          {error && <p className="status error">{error}</p>}
        </aside>

        <section className="panel files" aria-label="Generated files preview">
          <div className="panel-heading row">
            <div>
              <h2>Preview</h2>
              <p>{preview?.command?.join(' ') || 'Run a preview to see generated files.'}</p>
            </div>
            {selectedFile && (
              <div className="file-meta">
                <strong>{selectedFile.lines}</strong>
                <span>lines</span>
              </div>
            )}
          </div>

          <div className="preview-layout">
            <nav className="file-list" aria-label="Generated file list">
              {Object.entries(groupedFiles).map(([kind, files]) => (
                <div className="file-group" key={kind}>
                  <h3>{kind}</h3>
                  {files.map((file) => (
                    <button
                      key={file.path}
                      className={file.path === selectedFile?.path ? 'file-button active' : 'file-button'}
                      onClick={() => setSelectedPath(file.path)}
                    >
                      <span>{file.path}</span>
                      <small>{file.bytes} B</small>
                    </button>
                  ))}
                </div>
              ))}
            </nav>

            <article className="code-card">
              {selectedFile ? (
                <>
                  <div className="code-title">
                    <strong>{selectedFile.path}</strong>
                    <span>{selectedFile.kind}</span>
                  </div>
                  <pre>
                    <code>{selectedFile.content}</code>
                  </pre>
                </>
              ) : (
                <div className="empty-state">No preview has been generated yet.</div>
              )}
            </article>
          </div>
        </section>
      </section>
    </main>
  );
}

function labelFor(field) {
  return {
    frontend: 'Frontend',
    backend: 'Backend',
    database: 'Database',
    cache: 'Cache',
  }[field];
}

function display(value) {
  return {
    react: 'React + Vite',
    vue: 'Vue + Vite',
    go: 'Go API',
    node: 'Node API',
    python: 'Python API',
    php: 'PHP API',
    java: 'Java API',
    postgres: 'PostgreSQL',
    mysql: 'MySQL',
    redis: 'Redis',
    none: 'None',
  }[value] || value;
}

function addonLabel(value) {
  return {
    nginx: 'Nginx reverse proxy',
    localstack: 'LocalStack',
    monitoring: 'Prometheus / Grafana',
    cicd: 'GitHub Actions',
  }[value];
}

export default App;
