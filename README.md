# dev-box

![dev-box demo](https://raw.githubusercontent.com/Sousuke-Naritomi/dev-box/main/demo.gif)

`dev-box` は、フロントエンド、バックエンド、データベースなどを選択し、ローカル開発環境のひな形を生成する Go 製 CLI です。

対話形式とコマンドラインフラグの両方に対応し、Docker Compose、アプリケーションのスキャフォールド、GitHub Actions、Kubernetes マニフェスト、Terraform 設定を生成します。

> [!IMPORTANT]
> 本ツールが生成する Kubernetes と Terraform の設定は、開発・検証を始めるためのベースラインです。インターネット公開や本番運用の前に、イメージレジストリ、認証情報、ネットワーク制御、TLS、バックアップ、監視、コスト設定を利用環境に合わせて見直してください。

## 目次

- [提供機能](#提供機能)
- [機能要件](#機能要件)
- [非機能要件](#非機能要件)
- [対応スタック](#対応スタック)
- [前提条件](#前提条件)
- [インストール](#インストール)
- [クイックスタート](#クイックスタート)
- [コマンドリファレンス](#コマンドリファレンス)
- [生成物](#生成物)
- [Kubernetes の利用手順](#kubernetes-の利用手順)
- [Terraform の利用手順](#terraform-の利用手順)
- [CI/CD](#cicd)
- [セキュリティ](#セキュリティ)
- [制約と運用上の注意](#制約と運用上の注意)
- [開発とテスト](#開発とテスト)
- [アーキテクチャ](#アーキテクチャ)

## 提供機能

- 対話型プロンプトによる技術スタック選択
- 一部のフラグだけを指定し、不足項目だけを対話入力するハイブリッド操作
- Docker Compose とアプリケーションコードの同時生成
- Docker または Podman による開発環境の起動、停止、状態確認、ログ表示
- 暗号学的乱数を使用したデータベースおよび Grafana パスワード生成
- Nginx、LocalStack、監視スタック、GitHub Actions の任意追加
- Kubernetes の Deployment、Service、PVC、NetworkPolicy 生成
- AWS、GCP、Azure 向け Terraform テンプレート生成

## 機能要件

| ID | 要件 | 実装内容 |
|---|---|---|
| FR-01 | スタックを選択できる | frontend、backend、database、cache を選択可能 |
| FR-02 | 対話操作に対応する | 未指定項目のみ対話プロンプトで入力 |
| FR-03 | 非対話生成に対応する | 必須選択をフラグで指定し、CI 等から実行可能 |
| FR-04 | ローカル環境を生成する | `docker-compose.yml`、`.env`、ソースコード、Dockerfile を生成 |
| FR-05 | 既存ファイルを保護する | 既存ファイル検出時に確認し、`--force` 指定時のみ無条件で上書き |
| FR-06 | 開発環境を操作できる | `start`、`stop`、`status`、`logs` を提供 |
| FR-07 | オプション機能を追加できる | Nginx、LocalStack、監視、GitHub Actions を選択可能 |
| FR-08 | Kubernetes 設定を生成できる | Namespace、Deployment、Service、PVC、NetworkPolicy を生成 |
| FR-09 | Terraform 設定を生成できる | AWS、GCP、Azure 向け VM とネットワークのひな形を生成 |
| FR-10 | 秘密情報を分離する | `.env` と公開用 `.env.example` を別々に生成 |
| FR-11 | 動的ポート割り当てに対応する | ホスト側ポートを Docker Compose に自動割り当てさせ、`status` で確認可能 |

### 入力値

| 項目 | 選択肢 |
|---|---|
| frontend | `react`, `vue`, `none` |
| backend | `node`, `python`, `go`, `php`, `java`, `none` |
| database | `mysql`, `postgres`, `none` |
| cache | `redis`, `none` |
| optional | `nginx`, `localstack`, `monitoring`, `cicd` |

## 非機能要件

| ID | 分類 | 内容 |
|---|---|---|
| NFR-01 | 再現性 | 主要なコンテナイメージを固定バージョンで生成 |
| NFR-02 | セキュリティ | `crypto/rand` によるパスワード生成、`.env` の Git 除外、Kubernetes NetworkPolicy を提供 |
| NFR-03 | 可搬性 | Linux、macOS、Windows でビルド可能。Docker と Podman を自動検出 |
| NFR-04 | 保守性 | 設定、ファイルシステム、生成器、技術スタック定義をパッケージ分割 |
| NFR-05 | テスト容易性 | FileSystem と Generator をインターフェース化し、モックによる単体テストに対応 |
| NFR-06 | 障害解析 | 生成 CI は readiness 確認をリトライし、失敗時に対象サービスのログを出力 |
| NFR-07 | キャンセル | CLI は割り込みシグナルを Context に伝播し、ファイル生成中の中断に対応 |

## 対応スタック

| カテゴリ | 対応項目 | 既定イメージ |
|---|---|---|
| フロントエンド | React + Vite、Vue + Vite | `node:20-alpine` |
| バックエンド | Node.js + Express | `node:20-alpine` |
|  | Python + Flask | `python:3.12-slim` |
|  | Go + net/http | `golang:1.22-alpine` |
|  | PHP + Apache | `php:8.3-apache` |
|  | Java + HttpServer | `eclipse-temurin:17-jdk` |
| データベース | MySQL | `mysql:8.0` |
|  | PostgreSQL | `postgres:16-alpine` |
| キャッシュ | Redis | `redis:7-alpine` |
| リバースプロキシ | Nginx | `nginx:1.27-alpine` |
| AWS ローカル開発 | LocalStack | `localstack/localstack:3.5` |
| 監視 | Prometheus | `prom/prometheus:v2.53.0` |
|  | Grafana | `grafana/grafana:11.1.0` |
|  | Loki / Promtail | `grafana/loki:3.1.0`, `grafana/promtail:3.1.0` |

## 前提条件

### CLI のビルドまたはインストール

- Go 1.25 以上

### 生成したローカル環境の利用

- Docker Engine + Docker Compose V2、または Compose 互換の Podman

### オプション機能

- Kubernetes マニフェストの適用: Kubernetes クラスタと `kubectl`
- Terraform の実行: Terraform 1.0 以上と対象クラウドの認証情報
- GitHub Actions: GitHub リポジトリと Actions の実行権限

## インストール

```bash
go install github.com/Sousuke-Naritomi/dev-box@latest
```

ソースからビルドする場合:

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

## クイックスタート

### 対話形式

```bash
mkdir my-app
cd my-app
dev-box init
dev-box start
```

### 非対話形式

```bash
dev-box init \
  --frontend=react \
  --backend=go \
  --database=postgres \
  --cache=redis \
  --nginx \
  --cicd \
  --force

dev-box start
dev-box status
```

ホスト側ポートは動的に割り当てられます。接続先は `dev-box status` または次のコマンドで確認してください。

```bash
docker compose ps
docker compose port api 8080
```

## コマンドリファレンス

### `dev-box init`

ローカル開発環境とソースコードを現在のディレクトリへ生成します。

```text
--frontend     react | vue | none
--backend      node | python | go | php | java | none
--database     mysql | postgres | none
--cache        redis | none
--nginx        Nginx を追加
--localstack   LocalStack を追加
--monitoring   Prometheus、Grafana、Loki、Promtail を追加
--cicd         .github/workflows/ci.yml を追加
--force        確認せず既存ファイルを上書き
```

非対話環境では、4つの選択項目をすべて指定してください。未指定項目があると対話入力が必要になります。`--force` は上書き確認を省略するフラグであり、選択項目の指定を省略するものではありません。

```bash
dev-box init \
  --frontend=none \
  --backend=python \
  --database=postgres \
  --cache=none \
  --force
```

### `dev-box start`

Docker または Podman を検出し、`compose up -d` を実行します。起動後に割り当て済みポートを表示します。

### `dev-box stop`

`compose down` を実行します。永続ボリュームは削除しません。

### `dev-box status`

`compose ps` を実行し、コンテナ状態と公開ポートを表示します。

### `dev-box logs [service]`

ログを追跡表示します。

```bash
dev-box logs
dev-box logs api
dev-box logs frontend
```

### `dev-box k8s`

Kubernetes マニフェストを `k8s-manifests/` に生成します。

```bash
dev-box k8s \
  --frontend=react \
  --backend=go \
  --database=postgres \
  --cache=redis
```

### `dev-box deploy`

Terraform 設定を `terraform/` に生成します。

```bash
dev-box deploy --provider=aws
dev-box deploy --provider=gcp
dev-box deploy --provider=azure
```

## 生成物

選択内容に応じて、次のファイルが生成されます。

```text
.
├── docker-compose.yml
├── .env
├── .env.example
├── .gitignore
├── frontend/
│   ├── Dockerfile
│   └── ...
├── backend/
│   ├── Dockerfile
│   └── ...
├── nginx/
│   └── nginx.conf
├── monitoring/
│   ├── prometheus.yml
│   └── promtail-config.yml
└── .github/
    └── workflows/
        └── ci.yml
```

| 条件 | 生成物 |
|---|---|
| 常時 | `docker-compose.yml` |
| backend、database、monitoring のいずれかを選択 | `.env`, `.env.example`, `.gitignore` |
| frontend を選択 | `frontend/` |
| backend を選択 | `backend/` |
| `--nginx` | `nginx/nginx.conf` |
| `--monitoring` | `monitoring/` |
| `--cicd` | `.github/workflows/ci.yml` |

## Kubernetes の利用手順

`dev-box k8s` は次のリソースを生成します。

- Namespace: `dev-box-ns`
- frontend: Deployment 2 replicas、LoadBalancer Service
- api: Deployment 2 replicas、ClusterIP Service
- database: Deployment、ClusterIP Service、1 GiB PVC
- Redis: Deployment、ClusterIP Service
- ingress のデフォルト拒否と、frontend → api、api → db/cache の許可ルール

### アプリケーションイメージ

生成マニフェストは次のイメージ名を参照します。

- frontend: `dev-box-frontend:latest`
- backend: `dev-box-api:latest`

適用前に生成済み Dockerfile からイメージをビルドし、クラスタから取得可能なレジストリへ push するか、ローカルクラスタへ load してください。

```bash
docker build -t dev-box-frontend:latest ./frontend
docker build -t dev-box-api:latest ./backend
```

本番レジストリを使う場合は、`k8s-manifests/frontend.yaml` と `k8s-manifests/api.yaml` の `image` を変更してください。

### Database Secret

database を選択した場合、適用前に `db-secret` を作成します。

```bash
kubectl apply -f k8s-manifests/namespace.yaml
kubectl -n dev-box-ns create secret generic db-secret \
  --from-literal=password='replace-with-a-secure-password'
kubectl apply -f k8s-manifests/
```

NetworkPolicy の要件は CNI 実装や Ingress Controller により異なります。外部公開に必要な通信元が許可されるよう、生成されたポリシーをクラスタ構成に合わせて調整してください。

## Terraform の利用手順

Terraform テンプレートは、VM、基本ネットワーク、Docker Engine を準備します。アプリケーションの配置や起動は自動化しません。

```bash
dev-box deploy --provider=aws
cd terraform
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
terraform apply
```

| Provider | 主な生成対象 | 追加準備 |
|---|---|---|
| AWS | EC2、Security Group | AWS 認証情報、SSH key name |
| GCP | Compute Engine、Firewall | GCP 認証情報、project ID |
| Azure | Resource Group、VNet、NSG、Linux VM | Azure 認証情報、`~/.ssh/id_rsa.pub` |

生成テンプレートには開発用途の広い ingress 設定が含まれます。本番適用前に CIDR、SSH、HTTP/HTTPS、VM サイズ、ディスク、バックアップを見直してください。

## CI/CD

### 生成先プロジェクトの CI

`dev-box init --cicd` は `.github/workflows/ci.yml` を生成します。

生成ワークフローは次を実行します。

1. `.env.example` から CI 用 `.env` を準備
2. `docker compose config` による構文検証
3. Trivy による `CRITICAL` / `HIGH` のファイルシステムスキャン
4. `docker compose up -d` によるサービス起動
5. API または frontend の HTTP readiness 確認
6. readiness 失敗時のサービスログ出力
7. `docker compose down -v` による後片付け

`.env.example` の値は CI 動作確認用です。本番用の秘密情報には GitHub Actions Secrets などを使用してください。

### dev-box リポジトリ自身の CI

このリポジトリでは、GitHub Actions で次を実行します。

- `golangci-lint`
- `go test -v -cover ./...`

## セキュリティ

- database と Grafana のパスワードを `crypto/rand` で生成
- MySQL の root password と application password を分離
- `.env` を Git 管理対象から除外
- `.env.example` の秘密情報をプレースホルダへ置換
- `.env` を `0600` で作成するよう要求
- Nginx にレート制限、gzip、基本的なセキュリティヘッダを設定
- Kubernetes に resource requests/limits、readiness/liveness probe、NetworkPolicy を設定

Windows では POSIX のファイルモードがそのままアクセス制御リストへ対応しないため、`.env` の権限を必要に応じて別途確認してください。

## 制約と運用上の注意

- ホストポートは固定されません。起動ごとに `docker compose ps` で確認してください。
- `--force` は生成対象の既存ファイルを上書きします。既存プロジェクトへの適用前に Git へコミットしてください。
- すべてを `none` にし、オプションサービスも選択しない構成には実行対象サービスがありません。
- LocalStack は S3、SQS、SNS、DynamoDB を有効化します。
- monitoring はホストの `/var/log` を Promtail に read-only mount します。Docker Desktop や非 Linux 環境ではログ取得範囲が異なる場合があります。
- Kubernetes のアプリケーションイメージ名と Secret は利用者が準備する必要があります。
- Kubernetes の frontend Service は `LoadBalancer` です。ローカルクラスタでは追加のトンネルやロードバランサ実装が必要な場合があります。
- Terraform はインフラのひな形生成までを対象とし、アプリケーションデプロイ、DNS、TLS、監視、バックアップは対象外です。
- 生成コードは最小構成です。認証、認可、入力検証、DB migration、テスト、ログ設計などは用途に応じて追加してください。

## 開発とテスト

```bash
go test ./...
go vet ./...
```

Makefile を利用できる環境では次も使用できます。

```bash
make build
make test
make lint
```

リリース前の推奨確認:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./...
```

## アーキテクチャ

```text
dev-box/
├── cmd/                  # Cobra コマンドと CLI 制御
├── internal/
│   ├── config/           # 選択値、バージョン、バリデーション
│   ├── fs/               # OS / Mock ファイルシステム
│   ├── generator/        # Compose、CI、K8s、Terraform、コード生成
│   ├── registry/         # frontend / backend 定義
│   └── security/         # パスワード生成
├── main.go
├── Makefile
└── .goreleaser.yaml
```

主要な設計方針:

- 技術スタック固有情報を registry に集約
- テンプレートを `embed.FS` に同梱し、単一バイナリで配布可能
- FileSystem と Generator の抽象化によるテスト容易性
- 設定値を独自型で表現し、生成前にバリデーション

## ライセンス

[MIT License](LICENSE)
