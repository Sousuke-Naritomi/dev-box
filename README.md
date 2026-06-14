# dev-box

`dev-box` は、選択した技術スタックに応じてローカル開発環境のひな形を生成する Go 製 CLI です。

Docker Compose、フロントエンドとバックエンドの最小実装、Kubernetes マニフェスト、Terraform 構成、GitHub Actions ワークフローを生成できます。対話形式とコマンドラインフラグの両方に対応しています。

> [!IMPORTANT]
> 生成物は開発開始時のベースラインです。認証・認可、TLS、DNS、バックアップ、データベースマイグレーション、可用性、コスト管理などを含む完成済みの本番環境ではありません。本番利用前に、生成された設定を対象環境に合わせてレビューしてください。

## 目次

- [主な機能](#主な機能)
- [対応スタック](#対応スタック)
- [動作要件](#動作要件)
- [インストール](#インストール)
- [クイックスタート](#クイックスタート)
- [コマンド](#コマンド)
- [生成されるローカル開発環境](#生成されるローカル開発環境)
- [ファイル生成の安全性](#ファイル生成の安全性)
- [Kubernetes](#kubernetes)
- [Terraform](#terraform)
- [CI](#ci)
- [開発](#開発)
- [既知の制約](#既知の制約)

## 主な機能

- 対話式ウィザードによる技術スタック選択
- フラグを使用した非対話生成
- `dev-box.yaml`を使用した宣言的で再現可能な生成
- Docker Compose とアプリケーションコードの同時生成
- Docker または Podman を使用した起動、停止、状態確認、ログ表示
- React/Vue の本番ビルド用 Dockerfile 生成
- Node.js/Python/Go/PHP/Java の API サンプルと Dockerfile 生成
- MySQL/PostgreSQL、Redis、Nginx、LocalStack の任意追加
- Prometheus、Grafana、Loki、Promtail の監視構成生成
- Kubernetes の Deployment、Service、PVC、NetworkPolicy 生成
- AWS、GCP、Azure 向け Terraform 構成生成
- GitHub Actions ワークフロー生成
- 既存ファイル保護、dry-run、出力先指定、アトミックなファイル更新

## 対応スタック

### アプリケーション

| 種別 | 選択値 | 開発時のポート | 生成 Dockerfile のポート |
|---|---|---:|---:|
| Frontend | React + Vite | 5173 | 8080 |
| Frontend | Vue + Vite | 5173 | 8080 |
| Backend | Node.js + Express | 3000 | 8080 |
| Backend | Python + Flask | 8000 | 8080 |
| Backend | Go + `net/http` | 8080 | 8080 |
| Backend | PHP + Apache | 80 | 8080 |
| Backend | Java + `HttpServer` | 8080 | 8080 |

フロントエンドとバックエンドには `none` を指定できます。

すべてのバックエンドサンプルは次のエンドポイントを提供します。

| パス | 用途 |
|---|---|
| `/` | サンプルレスポンス |
| `/health` | ヘルスチェック |
| `/metrics` | Prometheus 形式の最小メトリクス |

### データストアとアドオン

| 種別 | 選択肢 |
|---|---|
| Database | MySQL 8.0、PostgreSQL 16 |
| Cache | Redis 7 |
| Reverse proxy | Nginx 1.27 |
| AWS emulator | LocalStack 3.5（S3、SQS、SNS、DynamoDB） |
| Monitoring | Prometheus、Grafana、Loki、Promtail |
| CI | GitHub Actions |

コンポーネントのバージョンは [`internal/config/versions.go`](internal/config/versions.go) で管理されています。現在はバージョンタグ固定であり、イメージ digest 固定ではありません。

## 動作要件

### dev-box のビルド

- Go 1.25 以上

### 生成したローカル環境の実行

- Docker Engine と Docker Compose V2、または Compose 互換の Podman

`start`、`stop`、`status`、`logs` は最初に `docker` を検索し、見つからない場合は `podman` を検索します。明示的に指定する場合は `DOCKER_CMD` 環境変数を使用します。

```bash
DOCKER_CMD=podman dev-box start
```

### オプション機能

- Kubernetes: `kubectl` と対象クラスタ
- Terraform: Terraform 1.0 以上、対象クラウドの認証情報
- コンテナイメージ検証: Docker デーモン

## インストール

### Go install

```bash
go install github.com/Sousuke-Naritomi/dev-box@latest
```

### ソースからビルド

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
go build -o dev-box .
```

Windows PowerShell:

```powershell
git clone https://github.com/Sousuke-Naritomi/dev-box.git
Set-Location dev-box
go build -o dev-box.exe .
```

バージョン情報は次のコマンドで確認できます。

```bash
dev-box version
```

リリースビルドでは CLI バージョンが埋め込まれます。通常の `go build` では `dev` と表示されます。

## クイックスタート

### 対話形式

空の作業ディレクトリで実行します。

```bash
mkdir my-app
cd my-app
dev-box init
dev-box start
dev-box status
```

### 非対話形式

非対話環境では `frontend`、`backend`、`database`、`cache` をすべて指定してください。

```bash
dev-box init \
  --frontend react \
  --backend go \
  --database postgres \
  --cache redis \
  --nginx \
  --monitoring \
  --cicd \
  --output-dir ./my-app
```

生成された`dev-box.yaml`を使用すれば、同じ構成を別ディレクトリへ再生成できます。

```bash
dev-box init --config ./my-app/dev-box.yaml --output-dir ./my-app-replayed
dev-box k8s --config ./my-app/dev-box.yaml --output-dir ./k8s-manifests
```

CLIで明示したフラグは設定ファイルより優先されます。

```bash
dev-box init --config dev-box.yaml --backend python --monitoring=false
```

ホスト側ポートは自動割り当てされ、`127.0.0.1` のみに公開されます。接続先は次のコマンドで確認します。

```bash
cd my-app
dev-box status
docker compose port frontend 5173
docker compose port api 8080
```

バックエンド種別によって API のコンテナポートは異なります。正確な値は生成された `docker-compose.yml` または `docker compose ps` で確認してください。

## コマンド

### `dev-box init`

Docker Compose、環境変数、アプリケーションコード、任意の監視・CI 設定を生成します。

```text
--frontend string    react | vue | none
--backend string     node | python | go | php | java | none
--database string    mysql | postgres | none
--cache string       redis | none
--nginx              Nginx reverse proxyを追加
--localstack         LocalStackを追加
--monitoring         Prometheus、Grafana、Loki、Promtailを追加
--cicd               GitHub Actionsを追加
--config string      dev-box.yamlのパス
--force              確認せず既存ファイルを上書き
--dry-run            ファイルを書き込まず生成内容を標準出力へ表示
--output-dir string  出力先ディレクトリ（既定値: .）
```

サービスが1つも存在しない構成は拒否されます。LocalStackまたはMonitoringだけを選択した構成は生成できます。

### `dev-box start`

カレントディレクトリで `docker compose up -d` または `podman compose up -d` を実行します。

```bash
dev-box start
```

### `dev-box stop`

`compose down` を実行します。`-v` は付与しないため、名前付きボリュームは保持されます。

```bash
dev-box stop
```

### `dev-box status`

`compose ps` を実行し、コンテナ状態と割り当て済みポートを表示します。

```bash
dev-box status
```

### `dev-box logs [service]`

全サービスまたは指定した1サービスのログを追跡表示します。

```bash
dev-box logs
dev-box logs api
```

サービス名は最大1件指定できます。

### `dev-box k8s`

選択した構成に対応する Kubernetes マニフェストを生成します。

```bash
dev-box k8s \
  --frontend react \
  --backend go \
  --database postgres \
  --cache redis \
  --output-dir ./k8s-manifests
```

`init`が生成した設定をそのまま利用する場合:

```bash
dev-box k8s --config ./dev-box.yaml
```

利用可能な共通フラグ:

```text
--force
--dry-run
--config string
--output-dir string  既定値: k8s-manifests
```

### `dev-box deploy`

AWS、GCP、Azure のいずれかに対応する Terraform ファイルを生成します。

```bash
dev-box deploy --provider aws --output-dir ./terraform
```

```text
--provider string    aws | gcp | azure
--force
--dry-run
--output-dir string  既定値: terraform
```

### `dev-box version`

CLI とテンプレートスキーマのバージョンを表示します。

```text
dev-box <version>
template schema 1
```

## 生成されるローカル開発環境

構成に応じて次のファイルが生成されます。

```text
my-app/
|-- dev-box.yaml
|-- docker-compose.yml
|-- .env
|-- .env.example
|-- .gitignore
|-- frontend/
|   |-- Dockerfile
|   |-- .dockerignore
|   |-- package.json
|   |-- package-lock.json
|   `-- src/
|-- backend/
|   |-- Dockerfile
|   |-- .dockerignore
|   `-- ...
|-- nginx/
|   `-- nginx.conf
|-- monitoring/
|   |-- prometheus.yml
|   |-- promtail-config.yml
|   `-- grafana/provisioning/datasources/prometheus.yml
`-- .github/workflows/ci.yml
```

選択していないコンポーネントのファイルやサービスは生成されません。

### `dev-box.yaml`

`init`は解決済みの構成を、バージョン付きの設定ファイルとして保存します。

```yaml
version: 1
frontend: react
backend: go
database: postgres
cache: redis
nginx: true
localstack: false
monitoring: true
cicd: true
```

- 未知のキーはエラーとして拒否
- 未対応の設定バージョンは拒否
- `--force`など実行時だけのオプションは保存しない
- CLIフラグで一部だけ上書き可能
- ComposeとKubernetesを同じ構成から再生成可能

### Docker Compose の仕様

- すべての公開ポートを `127.0.0.1:0:<container-port>` 形式でバインド
- ホスト側ポートを Compose によって動的割り当て
- アプリケーションコードを bind mount し、開発用サーバーを起動
- Node.js は `npm ci` を使用
- DB、Redis にヘルスチェックを設定
- DB、Node.js、Go、LocalStack、監視データに名前付きボリュームを使用
- MySQL/PostgreSQL、Grafana の秘密情報を `.env` から読み込み
- Nginx、LocalStack、監視サービスも localhost のみに公開

生成 Dockerfile は Compose の開発起動とは別用途です。React/Vue は Vite でビルドした静的ファイルを `nginx-unprivileged` で配信し、GoとJavaはマルチステージビルドを使用します。

### 秘密情報

データベースとGrafanaを選択した場合、暗号学的乱数を使用してパスワードを生成します。

- `.env`: 実際の秘密情報。可能な環境では `0600` で作成
- `.env.example`: プレースホルダー化した共有用設定
- `.gitignore`: 既存内容を保持したまま必要項目を追記

WindowsではPOSIXファイルモードがNTFS ACLへ直接対応しないため、`.env`のアクセス権を別途確認してください。

### 監視

`--monitoring` を指定すると次を生成します。

- Prometheus自身とAPIの`/metrics`を収集する設定
- GrafanaへPrometheusを自動登録するdatasource provisioning
- LokiとPromtailによる`/var/log/*.log`の収集設定

PromtailのホストログマウントはLinuxを前提としています。Docker Desktopでは取得できるログ範囲が異なります。

## ファイル生成の安全性

`init`、`k8s`、`deploy`は共通の安全な書き込み処理を使用します。

- 上書き対象を事前に一覧表示
- 対話端末では上書き確認を実施
- 非対話環境で既存ファイルがある場合は`--force`なしでは失敗
- `--dry-run`で書き込み前に内容を確認可能
- シンボリックリンクへの上書きを拒否
- 同一ディレクトリ内の一時ファイルへ書き込み後にrename
- 更新前のファイルを一時バックアップし、途中失敗時にロールバック
- 最終ファイルへ指定された権限を再適用
- `.gitignore`は置換せず、未登録行のみ追加
- 解決済み構成を`dev-box.yaml`として保存
- Ctrl+CなどのキャンセルをContext経由で処理

既存ファイルを上書きする場合は、事前にGitへコミットすることを推奨します。

## Kubernetes

### 生成物

```text
k8s-manifests/
|-- namespace.yaml
|-- networkpolicy.yaml
|-- frontend.yaml  # frontend選択時
|-- api.yaml       # backend選択時
|-- db.yaml        # database選択時
`-- cache.yaml     # redis選択時
```

生成されるリソース:

- Namespace: `dev-box-ns`
- Application: Deployment、Service、readiness/liveness probe
- Frontend Service: `LoadBalancer`
- API Service: `ClusterIP`
- Database: Deployment、ClusterIP Service、1 GiB PVC
- Redis: Deployment、ClusterIP Service
- 構成に応じて生成されるNetworkPolicy

### セキュリティ設定

フロントエンド、API、Redisには次の設定を生成します。

- `runAsNonRoot: true`
- 固定UIDによる`runAsUser`
- `allowPrivilegeEscalation: false`
- Linux capabilityの全削除
- `seccompProfile.type: RuntimeDefault`
- CPU/メモリのrequestsとlimits

MySQL/PostgreSQLは公式イメージの起動要件を優先しており、同じ固定UID設定は適用していません。

NetworkPolicyはデフォルトでIngress/Egressを拒否し、DNS、frontendからAPI、APIからDB/Redisへの必要な通信を追加します。frontendを選択した場合はIngress Controller名前空間および外部IPからfrontendへの通信を許可します。クラスタのCNI、DNSラベル、Ingress Controller名前空間に合わせて調整してください。

### 適用手順

アプリケーション用イメージをビルドし、クラスタから取得可能なレジストリへpushするか、ローカルクラスタへloadします。

```bash
docker build -t dev-box-frontend:latest ./frontend
docker build -t dev-box-api:latest ./backend
```

必要に応じて`frontend.yaml`と`api.yaml`のイメージ名を変更します。

データベースを選択した場合は、マニフェスト適用前にSecretを作成します。

```bash
kubectl apply -f k8s-manifests/namespace.yaml
kubectl -n dev-box-ns create secret generic db-secret \
  --from-literal=password='replace-with-a-secure-password'
kubectl apply -f k8s-manifests/
```

## Terraform

`deploy`は選択したプロバイダ向けに次を生成します。

```text
terraform/
|-- main.tf
|-- variables.tf
`-- terraform.tfvars.example
```

### 共通方針

- SSH許可CIDRの既定値は空
- HTTP/HTTPS公開は既定で無効
- AWS/GCPのPublic IPは既定で無効
- AWSのoutbound CIDRは既定で空
- `terraform.tfvars.example`には置換用のサンプルCIDRを記載
- VM起動時にDocker EngineとCompose pluginをインストール

### プロバイダ別設定

| Provider | 主なリソース | セキュリティ設定 |
|---|---|---|
| AWS | EC2、Security Group | IMDSv2必須、暗号化root disk、Public IP/SSH/Web/egress制御 |
| GCP | Compute Engine、Firewall | OS Login、Shielded VM、Public IP/SSH/Web制御 |
| Azure | Resource Group、VNet、NSG、Linux VM | Secure Boot、vTPM、公開鍵変数、SSH/Web CIDR制御 |

### 利用手順

```bash
dev-box deploy --provider aws
cd terraform
cp terraform.tfvars.example terraform.tfvars
```

`terraform.tfvars`のサンプル値を実環境の値へ変更してから実行します。

```bash
terraform fmt -recursive
terraform init
terraform validate
terraform plan
terraform apply
```

生成されるTerraformはVMと基本ネットワークを構築し、Dockerをインストールするところまでを対象とします。アプリケーションの転送、起動、TLS、DNS、監視、バックアップは行いません。

## CI

### dev-boxリポジトリ自身のCI

`.github/workflows/ci.yml`では次を検証します。

- `gofmt`
- `go mod tidy`後の差分
- `go vet`
- `go test -race`
- `golangci-lint`
- 複数スタックの`docker compose config`
- `dev-box.yaml`からの再生成と構成の安定性
- Kubernetes YAMLの`kubeconform`
- frontendと全バックエンドのDocker image build、非rootユーザー、HTTP health check
- AWS/GCP/Azureの`terraform fmt`、`init`、`validate`
- TrivyによるTerraform構成スキャン

### 生成先プロジェクトのCI

`init --cicd`で生成されるワークフローは次を実行します。

- Compose設定の検証
- Trivyによる`CRITICAL`/`HIGH`脆弱性スキャン
- `docker compose up -d`によるサービス起動
- コンテナ状態とhealth checkの待機
- APIまたはfrontendへのHTTP確認
- 失敗時のログ出力
- `docker compose down -v`による後片付け

## 開発

### ビルドとテスト

```bash
go mod tidy
go test ./...
go vet ./...
go build ./...
golangci-lint run ./...
```

Makefileを使用する場合:

```bash
make build
make test
make lint
```

### ディレクトリ構成

```text
dev-box/
|-- cmd/                  # Cobraコマンド、生成ファイルの書き込み制御
|-- internal/
|   |-- config/           # 選択値、バージョン、バリデーション
|   |-- fs/               # OS/Mockファイルシステム
|   |-- generator/        # Compose、K8s、Terraform、CI、scaffold生成
|   |-- registry/         # frontend/backend定義とポート情報
|   `-- security/         # パスワード生成
|-- frontend/             # リポジトリ自身の生成例
|-- backend/              # リポジトリ自身の生成例
|-- k8s-manifests/        # リポジトリ自身の生成例
|-- main.go
|-- Makefile
`-- .goreleaser.yaml
```

テンプレートは`embed.FS`に組み込まれるため、リリース後は単一バイナリで生成できます。

## 既知の制約

- Composeのホスト側ポートは固定されません。起動後に`status`または`docker compose port`で確認してください。
- Composeは開発用のbase imageとbind mountを使用します。生成Dockerfileは主に本番イメージ/Kubernetes向けです。
- Kubernetesのfrontend/APIイメージ名は`dev-box-frontend:latest`と`dev-box-api:latest`です。利用者がビルド、配布、タグ管理を行う必要があります。
- KubernetesのDatabaseはStatefulSetではなくDeploymentとPVCです。本番データベース用途ではマネージドサービスや専用Operatorを検討してください。
- Kubernetesの`db-secret`は自動生成されません。
- frontend用NetworkPolicyには`0.0.0.0/0`からのIngress許可が含まれます。非公開構成では削除またはCIDR制限してください。
- PromtailはGrafana Labsの今後のサポート方針を確認し、必要に応じてGrafana Alloyなどへの移行を検討してください。
- LocalStackは選択したAWSサービスを起動しますが、リソース自体は作成しません。
- AWS/GCPはPublic IPを既定で作成しません。外部アクセスや起動時のパッケージ取得には、Public IPの明示的な有効化、NAT、踏み台、IAPなどを環境に合わせて構成してください。Azureは現行テンプレートでPublic IPを作成します。
- コンテナイメージはバージョンタグ固定であり、digest固定ではありません。
- 生成コードは最小実装です。認証、認可、入力検証、永続化、migration、業務ロジック、包括的テストは含まれません。

## ライセンス

[MIT License](LICENSE)
