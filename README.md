# dev-box

dev-boxは、ローカル開発環境の構築から本番環境向けのマニフェストおよびインフラストラクチャ・アズ・コード（IaC）リソースの生成までを自動化するためのコマンドライン・インターフェース（CLI）ツールです。Go言語で開発されており、単一のバイナリとして動作します。

本ツールは、対話式プロンプトまたはコマンドライン引数による設定に基づき、フルスタック・アプリケーションの土台となるボイラープレート、Docker Compose設定、Kubernetesマニフェスト、およびTerraform構成ファイルを安全かつ再現可能な形で生成します。

## 1. 概要とアーキテクチャ

dev-boxが生成する環境は、単なるコードの雛形ではなく、本番環境を見据えたエンタープライズ・グレードのアーキテクチャ設計とセキュリティ基準に準拠しています。

### 1.1. 関心の分離（Port Separation）
ローカル開発環境とKubernetes配備時のプロセス実行環境の要件の違いを吸収するため、各コンテナにおけるポート設計を明確に分離しています。
*   **InternalPort**: ローカル開発時（Docker Compose）に使用されます。各フレームワークの標準ポート（Viteの5173、Expressの3000等）を維持し、ホットリロードやデバッグの利便性を損ないません。
*   **ImagePort**: Kubernetesでの稼働時（Dockerイメージビルド後）に使用されます。マルチステージビルドを経て生成される本番向けコンテナは、一律で非特権ポート（8080等）を公開するように設計されています。

### 1.2. セキュリティと権限最小化
Kubernetesマニフェスト（YAML）およびDockerfileは、セキュリティのベストプラクティスに従い厳格に設計されています。
*   **非特権ユーザーによる実行**: すべてのバックエンドおよびフロントエンドコンテナは、ビルドプロセス内で専用の非rootユーザー（UID 10001, 101等）を作成し、`USER`ディレクティブを通じて実行権限を最小化しています。
*   **コンテナの堅牢化（Hardening）**: Kubernetes Deploymentマニフェストにおいて、`runAsNonRoot: true`、`allowPrivilegeEscalation: false`、`capabilities: drop: ["ALL"]` を強制し、コンテナの特権昇格や不要なシステムコールを遮断します。
*   **seccompプロファイルの適用**: `seccompProfile: { type: RuntimeDefault }` を指定し、ホストカーネルへのアクセスを制限しています。

### 1.3. ゼロトラスト・ネットワーク
Kubernetes環境において、ネームスペース内の通信はデフォルトで遮断されます（Default Deny）。
dev-boxが生成するNetworkPolicyマニフェストは、以下の通り必要な通信のみを明示的に許可するホワイトリスト方式を採用しています。
*   フロントエンドからバックエンドAPIへのEgress許可
*   バックエンドAPIからデータベースおよびキャッシュ（Redis）へのEgress許可
*   全Podからkube-dns（ポート53 UDP/TCP）へのDNS解決のためのEgress許可
*   Ingress ControllerからフロントエンドへのIngress許可

## 2. インストール手順

Go言語がインストールされている環境で、以下のコマンドを実行してビルドを行います。

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
go build -o dev-box .
```

ビルドされた `dev-box` バイナリを実行可能なパスに配置してください。

## 3. サポートされているテクノロジースタック

dev-boxは以下の技術スタックの組み合わせをサポートしています。

### フロントエンド (Frontend)
*   React (Vite)
*   Vue.js (Vite)
*   none (フロントエンドなし)

### バックエンド (Backend)
*   Go
*   Python (Flask)
*   Node.js (Express)
*   Java (HttpServer)
*   PHP (Apache)
*   none (バックエンドなし)

### データベース (Database)
*   PostgreSQL
*   MySQL
*   none (データベースなし)

### キャッシュ・メッセージブローカー (Cache)
*   Redis
*   none

### アドオン (Addons)
*   **Nginx Reverse Proxy**: バックエンドとフロントエンドへのルーティングを統括するAPIゲートウェイ（Docker Compose環境用）。
*   **LocalStack**: AWSリソースのローカルモック環境。
*   **Monitoring**: Prometheus、Grafana、およびPromtail/Lokiスタックによる監視とログ集約環境。
*   **CI/CD**: GitHub Actions用のCIワークフロー構成ファイル。

## 4. コマンドライン・リファレンス

dev-boxは、用途に応じた複数のサブコマンドを提供します。

### 4.1. 初期化コマンド: `dev-box init`
現在のディレクトリ（または指定したディレクトリ）に開発環境を構築します。引数なしで実行した場合は、インタラクティブなウィザードが起動します。

**フラグによる非対話的実行の例:**
```bash
dev-box init \
  --frontend react \
  --backend go \
  --database postgres \
  --cache redis \
  --nginx \
  --monitoring \
  --cicd \
  --output-dir ./my-project
```

**主要オプション:**
*   `--dry-run`: 実際のファイルシステムへの書き込みを行わず、生成されるファイル群と内容を出力します。
*   `--force`: 既存のファイルと競合が発生した場合でも、警告を出さずに上書きします。
*   `--output-dir`: ファイルを生成するターゲットディレクトリを指定します（デフォルトはカレントディレクトリ）。

### 4.2. 環境操作コマンド
ローカル環境のDocker Composeライフサイクルを管理するラッパーコマンド群です。

*   `dev-box start`: `docker compose up -d` をバックグラウンドで実行し、生成されたコンテナ群を起動します。
*   `dev-box stop`: `docker compose down` を実行し、環境を停止・削除します。
*   `dev-box status`: `docker compose ps` を実行し、コンテナの稼働状況を表示します。
*   `dev-box logs`: `docker compose logs -f` を実行し、すべてのコンテナの標準出力をストリーム表示します。

### 4.3. Kubernetesマニフェスト生成: `dev-box k8s`
選択したスタックに適合するKubernetesマニフェスト一式（YAMLファイル）を生成します。

```bash
dev-box k8s --frontend react --backend go --database postgres --cache redis
```

**生成されるファイル（例）:**
*   `namespace.yaml`: アプリケーションを隔離するためのネームスペース定義。
*   `api.yaml`: バックエンドのDeploymentおよびClusterIP Service。
*   `frontend.yaml`: フロントエンドのDeploymentおよびLoadBalancer Service。
*   `db.yaml`: データベースのStatefulSet（またはDeployment）およびPersistentVolumeClaim。
*   `cache.yaml`: RedisキャッシュのDeployment。
*   `networkpolicy.yaml`: 前述のゼロトラスト・セキュリティに基づくきめ細やかなトラフィック制御ポリシー。

### 4.4. クラウド・デプロイメント生成: `dev-box deploy`
インフラ構築を自動化するためのTerraformコード（IaC）を生成します。

```bash
dev-box deploy --provider aws
```

**サポートされるプロバイダ:**
*   `aws`: Amazon Web Services 向けモジュール
*   `gcp`: Google Cloud Platform 向けモジュール
*   `azure`: Microsoft Azure 向けモジュール

## 5. 自動生成される構成ファイルの詳細

### 5.1. クレデンシャル管理 (`.env` および `.gitignore`)
プロジェクトの初期化時、セキュアなパスワード（20文字の英数字および記号）が暗号論的疑似乱数生成器（CSPRNG）を用いて自動生成され、`.env`ファイルに書き込まれます。同時に、この`.env`ファイルは自動的に`.gitignore`に追記され、リポジトリへの偶発的なコミットを防止します。

### 5.2. Dockerfileの構造
すべてのDockerfileは、ビルド環境と実行環境を分離したマルチステージビルドを採用しています。
ビルドステージで依存関係の解決とコンパイルを行い、最終イメージには実行に必要なバイナリや静的ファイルのみをコピーすることで、イメージサイズの削減と攻撃表面（Attack Surface）の極小化を実現しています。

### 5.3. 監視・オブザーバビリティ
`--monitoring` フラグを有効にした場合、以下が自動設定されます。
*   **Prometheus**: バックエンドアプリケーションの `/metrics` エンドポイントをスクレイプするように構成された `prometheus.yml`。
*   **Grafana**: PrometheusおよびLokiをデータソースとして自動登録するためのプロビジョニング設定。
*   **Loki / Promtail**: Dockerのログドライバーからログを収集し、統合的な可視化環境を提供。

## 6. ライセンスと貢献について
開発に参加する場合、リポジトリをフォークした上でFeature Branchを作成し、Pull Requestを通じてコードを提案してください。新規にサブコマンドを追加する場合や、ジェネレータのテンプレート（`internal/generator/templates` 以下）を修正する場合は、必ず既存のGo言語ユニットテストおよびE2Eテストがパスすることを確認してください。
