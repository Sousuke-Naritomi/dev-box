# dev-box 🚀

`dev-box` は、ローカル開発環境から本番環境へのデプロイ（Docker Compose、Kubernetes、Terraform）まで、セキュアで即座に使える環境を自動生成する高速で柔軟なGo言語製のCLIツールです。

## ✨ 主な機能

- **対話型スカフォールド**: CLIの対話形式、またはコマンドライン引数を使って、フルスタックプロジェクト（フロントエンド ＋ バックエンド ＋ データベース）を瞬時に構築します。
- **サポートしているフレームワーク**:
  - **フロントエンド**: React (Vite), Vue.js (Vite)
  - **バックエンド**: Node.js (Express), Python (Flask), Go, PHP (Apache), Java (HttpServer)
  - **データベース**: PostgreSQL, MySQL
  - **キャッシュ**: Redis
- **高度なアーキテクチャ設計とセキュリティ**:
  - ローカル開発用のポート（`InternalPort`：ホットリロード対応）と、Kubernetes 運用時のポート（`ImagePort`）を分離したマルチステージ Dockerfile を自動生成。
  - すべての言語において、特権を持たない非rootコンテナ（`10001` や `101` などの標準ユーザー）として実行される設計。
- **本番環境水準の Kubernetes 対応**:
  - `dev-box k8s` コマンドで、`Deployment`, `Service`, `PVC` マニフェストを生成。
  - デフォルトで強固なセキュリティ設定: `seccompProfile` の適用、全 Capability のドロップ、非 root 実行 (`RunAsUser`) を強制。
  - ゼロトラストネットワーク: 厳格な Ingress / Egress 制御を行う `NetworkPolicy` を自動生成。
- **クラウドインフラ (IaC) 生成**: `dev-box deploy` で、AWS、GCP、Azure向けの Terraform モジュールを生成。
- **組み込みアドオン**:
  - 🔄 **Nginx リバースプロキシ**: ローカル開発時の API ルーティングを自動化。
  - ☁️ **LocalStack**: AWSのローカルモック環境。
  - 📈 **オブザーバビリティ (監視)**: Prometheus, Grafana, Promtail/Loki の設定を事前構成。
  - 🤖 **CI/CD**: 自動テストやビルド用の GitHub Actions ワークフローを生成。

---

## 📦 インストール方法

*(Go言語がインストールされている必要があります)*

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
go build -o dev-box .
```

生成されたバイナリを `PATH` の通ったディレクトリに移動してご利用ください。

---

## 🚀 使い方

### 1. ローカル環境の初期化 (`dev-box init`)

対話モード、またはフラグを指定して `docker-compose.yml`、ベースコード、`.env`、Nginx 設定などを生成します。

```bash
# 対話モード
./dev-box init

# 非対話モード（CI/CDなどで便利です）
./dev-box init --frontend react --backend go --database postgres --monitoring --cicd
```

**`init` で便利なフラグ**:
- `--output-dir <dir>`: カレントディレクトリではなく、指定したディレクトリにファイルを生成します。
- `--dry-run`: 実際にファイルを書き込まずに、生成される内容をプレビューします。
- `--force`: 既存のファイルがある場合でも、確認プロンプトをスキップして上書きします。

### 2. ローカル環境の操作

CLIから直接 Docker Compose 環境を操作できます：

```bash
./dev-box start    # docker compose up -d と同等
./dev-box stop     # docker compose down と同等
./dev-box status   # docker compose ps と同等
./dev-box logs     # docker compose logs -f と同等
```

### 3. Kubernetes マニフェストの生成 (`dev-box k8s`)

現在のスタック構成を元に、強固な Kubernetes YAML ファイルをエクスポートします（デフォルトでは `k8s-manifests/` フォルダに出力されます）。

```bash
./dev-box k8s --frontend react --backend go --database postgres --cache redis
```
これにより、`api.yaml`, `frontend.yaml`, `db.yaml`, `cache.yaml`, `namespace.yaml` に加えて、極めて堅牢な `networkpolicy.yaml` が生成されます。

### 4. Terraform クラウドインフラの生成 (`dev-box deploy`)

コンテナをクラウドにデプロイするための Terraform 構成を生成します。

```bash
./dev-box deploy --provider aws
```

---

## 🔒 実装されているベストプラクティスとセキュリティ

`dev-box` は単なる「Hello World」のボイラープレートではなく、エンタープライズの現場で通用するセキュリティに重きを置いています：
- **クレデンシャルの自動生成**: 初期化時に暗号論的疑似乱数を用いた20文字の `.env` ファイルを生成し、自動的に `.gitignore` へ追加します。
- **ゼロトラストな Kubernetes ネットワーク**: 明示的に許可されない限り、Pod同士やインターネットへの通信はできません（例：APIからDBへの通信は許可されるが、フロントエンドからDBへの直接通信は遮断）。
- **ディストロレス＆マルチステージビルド**: GoはAlpine上で、React/Vueは非特権Nginx(`nginx-unprivileged`)上で動かすなど、最小限のフットプリントとセキュアな基盤で実行されます。
- **ファイルシステムの保護**: シンボリックリンクの上書きを拒否し、Windows環境ではクレデンシャルファイルの NTFS パーミッションに関する警告を出力します。

## 🤝 コントリビュート
コントリビュート、Issueの報告、機能リクエストは大歓迎です！ぜひ Issues ページをご確認ください。
