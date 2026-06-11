# dev-box

![dev-box demo](https://raw.githubusercontent.com/Sousuke-Naritomi/dev-box/main/demo.gif)

Go言語で書かれた、モダンな開発環境を対話的に生成・管理する強力なCLIツールです。

Docker Compose、Kubernetesマニフェスト、Terraform設定からソースコードのスキャフォールドまで、ターミナル上で直感的に選択するだけで瞬時に構築します。チーム内での環境構築の手間を省き、ベストプラクティスに沿ったセキュアな構成を標準化します。

---

## ✨ 主な特徴

- 🎯 **対話型＆コマンドライン両対応**: `huh` を用いたリッチな対話型プロンプト。一部のフラグだけ（例: `--frontend=react`）を指定した場合、**不足している項目だけを動的に質問**する洗練されたUX。
- 🛡 **型安全＆アーキテクチャの洗練**: `config.FrontendType` のような強い型付けと、DI（依存性の注入）を用いた疎結合な設計（Mock FS / Mock Generator）。
- 🧩 **拡張性の高い Registry パターン**: 対応言語やフレームワークは `internal/registry` で一元管理。K8sジェネレーター等のハードコードを排除し、**新言語追加時に既存コードの修正が不要な完全な OCP（開放閉鎖の原則）** を実現。
- 🔒 **セキュア・バイ・デフォルト**: `crypto/rand` を用いた強固なパスワード自動生成、`.env` の `0600` 権限設定、DBのRoot/Appパスワードの完全分離、K8sの `NetworkPolicy` によるデフォルト・デナイ設定。
- 🐳 **Podman 対応**: Docker デーモンが見つからない場合、自動的に Podman を検知してシームレスにフォールバック（`dev-box status`, `dev-box logs` 等に対応）。

---

## 📦 前提条件

| ツール | バージョン |
|--------|-----------|
| Go | 1.25 以上 |
| Docker または Podman | 最新版（Compose V2 互換） |
| Terraform | 1.0 以上（`deploy` コマンド使用時） |

## 🚀 インストール

```bash
go install github.com/Sousuke-Naritomi/dev-box@latest
```

または、リポジトリをクローンしてビルド：

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
make build
```

## 🛠 使い方

### 1. 開発環境の初期化 (`init`)

```bash
# 全て対話プロンプトで進める
dev-box init

# フロントエンドだけ指定し、残りをプロンプトで答える（最高のUX）
dev-box init --frontend=react

# CI/CDなどで完全に非対話で生成する
dev-box init --frontend=react --backend=go --database=postgres --cache=redis --force
```

生成された環境の起動と停止：
```bash
dev-box start   # docker compose up -d (Podman自動フォールバック対応)
dev-box stop    # docker compose down
dev-box status  # コンテナのステータス確認
dev-box logs    # ログのストリーミング表示
```

### 2. Kubernetes マニフェスト生成 (`k8s`)

```bash
dev-box k8s --backend=go
```
`k8s-manifests/` ディレクトリに、Deployment, Service, PVC, NetworkPolicy などのマニフェストを出力します。イメージやポート番号は Registry から動的に解決されます。

### 3. Terraform 設定生成 (`deploy`)

```bash
dev-box deploy
```
AWS, GCP, Azure 向けの `main.tf` と `variables.tf` を対話的に生成します。

---

## 📚 対応技術スタック

| カテゴリ | 対応項目 |
|---------|---------|
| **フロントエンド** | React (Vite), Vue.js (Vite) |
| **バックエンド** | Node.js (Express), Python (Flask), Go (net/http), PHP (Apache), Java (HttpServer) |
| **データベース** | MySQL (8.0), PostgreSQL (16-alpine) |
| **キャッシュ** | Redis (7-alpine) |
| **プロキシ** | Nginx (1.27-alpine) — レートリミット、gzip、WebSocket対応 |
| **AWS 開発** | LocalStack (3.5) — S3, SQS, SNS, DynamoDB |
| **オブザーバビリティ**| Prometheus (v2.53.0), Grafana (11.1.0), Loki (3.1.0), Promtail (3.1.0) |
| **CI/CD** | GitHub Actions（Trivy セキュリティスキャン、golangci-lint） |
| **インフラ** | Kubernetes, AWS, GCP, Azure (Terraform) |

※ 本番環境の再現性を担保するため、Dockerイメージのバージョンは `latest` を避け、全て具体的なバージョンにピン留めされています。

---

## 🏗 アーキテクチャ

本プロジェクトはプロフェッショナルな Go 開発のベストプラクティスに基づき設計されています。

```text
dev-box/
├── cmd/                  # Cobra を用いた CLI エントリポイント
├── internal/
│   ├── config/           # 強い型を用いた設定とバリデーション
│   ├── fs/               # ファイルシステム抽象化（モックテスト用DI）
│   ├── generator/        # Text/Template によるマニフェスト・コード生成
│   ├── registry/         # フレームワークや言語定義を一元管理（Registryパターン）
│   └── security/         # 暗号論的パスワード生成ロジック
├── main.go
└── Makefile
```

### ソフトウェア設計原則
- **Open-Closed Principle (OCP)**: 新しい言語を足す際は `internal/registry` のマップに定義を追加するだけで、ジェネレーター（Docker/K8s等）の既存コードを変更する必要がありません。
- **Dependency Injection (DI)**: `fs.FileSystem` や `generator.Generator` などのインターフェースを通じて依存を注入しており、I/Oを伴わない高速で堅牢な単体テストが可能です。

---

## 🧪 テストと品質

```bash
make test
```

- **カバレッジの向上**: コアロジック、セキュリティジェネレーター、CLIコマンドのE2Eテストまで、広範なテストカバレッジを維持。
- **Lint と静的解析**: `golangci-lint` および `go vet` による厳格なコード品質チェックをCIで自動化しています。

---

## 📄 ライセンス

[MIT License](LICENSE)
