import { useEffect, useMemo, useState } from 'react';
import './style.css';

const options = {
  frontend: ['react', 'vue', 'none'],
  backend: ['go', 'node', 'python', 'php', 'java', 'none'],
  database: ['postgres', 'mysql', 'none'],
  cache: ['redis', 'none'],
  cloud: ['none', 'aws', 'gcp', 'azure'],
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
  cloud: 'none',
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
      throw new Error(text || `リクエストに失敗しました: ${response.status}`);
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
      setMessage(`${data.message}。${data.fileCount}件のファイルを書き込みました。`);
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
          <p className="eyebrow">dev-box で構築</p>
          <h1>DevBox Studio</h1>
          <p className="hero-copy">
            スタックを選び、dev-box が作成するファイルを確認してから、ブラウザからすぐ動かせるプロジェクトを生成できます。
          </p>
        </div>
        <div className="hero-stats" aria-label="現在のプレビュー概要">
          <span>{preview?.files?.length || 0}</span>
          <small>生成予定ファイル</small>
        </div>
      </section>

      <section className="workspace">
        <aside className="panel controls" aria-label="構成選択">
          <div className="panel-heading">
            <h2>構成</h2>
            <p>プレビューはローカルの dev-box CLI から生成されます。</p>
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

          <div className="addons" aria-label="追加機能">
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
            <span>出力プロジェクト名</span>
            <input
              value={config.project}
              onChange={(event) => updateField('project', event.target.value)}
              placeholder="studio-sample"
            />
          </label>

          <div className="actions">
            <button className="primary" onClick={loadPreview} disabled={loading}>
              {loading ? 'プレビュー中...' : 'ファイルをプレビュー'}
            </button>
            <button className="secondary" onClick={generateProject} disabled={generating || loading}>
              {generating ? '生成中...' : 'プロジェクトを生成'}
            </button>
          </div>

          {preview?.summary && <p className="status good">{preview.summary}</p>}
          {message && <p className="status good">{message}</p>}
          {error && <p className="status error">{error}</p>}
        </aside>

        <section className="panel files" aria-label="生成ファイルのプレビュー">
          <div className="panel-heading row">
            <div>
              <h2>プレビュー</h2>
              <p>{preview?.command?.join(' ') || 'プレビューを実行すると生成予定ファイルが表示されます。'}</p>
            </div>
            {selectedFile && (
              <div className="file-meta">
                <strong>{selectedFile.lines}</strong>
                <span>行</span>
              </div>
            )}
          </div>

          <div className="preview-layout">
            <nav className="file-list" aria-label="生成ファイル一覧">
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
                <div className="empty-state">まだプレビューは生成されていません。</div>
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
    frontend: 'フロントエンド',
    backend: 'バックエンド',
    database: 'データベース',
    cache: 'キャッシュ',
    cloud: 'クラウド (Terraform)',
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
    aws: 'AWS',
    gcp: 'Google Cloud',
    azure: 'Azure',
    none: 'なし',
  }[value] || value;
}

function addonLabel(value) {
  return {
    nginx: 'Nginx リバースプロキシ',
    localstack: 'LocalStack',
    monitoring: 'Prometheus / Grafana 監視',
    cicd: 'GitHub Actions',
  }[value];
}

export default App;