# dev-box

![dev-box demo](https://raw.githubusercontent.com/Sousuke-Naritomi/dev-box/main/demo.gif)

Go言語で書かれた、開発環境の構成ファイルを対話的に生成するCLIツールです。

Docker Compose、Kubernetes マニフェスト、Terraform 設定などを、ターミナル上で質問に答えるだけで生成できます。環境構築の手間を減らし、チーム内の構成を標準化することを目的としています。

## 前提条件

| ツール | バージョン |
|--------|-----------|
| Go | 1.22 以上 |
| Docker | 24.0 以上（Compose V2 プラグイン含む）|
| Terraform | 1.0 以上（`deploy` コマンド使用時）|

## インストール

```bash
go install github.com/Sousuke-Naritomi/dev-box@latest
```

または、リポジトリをクローンしてビルド：

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
make build
```

## 使い方

### 開発環境の初期化

```bash
dev-box init
```

対話形式で以下を選択し、`docker-compose.yml` と関連ファイルを生成します。

```bash
dev-box start   # docker compose up -d
dev-box stop    # docker compose down
```

### Kubernetes マニフェスト生成

```bash
dev-box k8s
```

`k8s-manifests/` ディレクトリに Deployment、Service、PersistentVolumeClaim の YAML を生成します。

### Terraform 設定生成

```bash
dev-box deploy
```

`terraform/` ディレクトリに `main.tf` と `variables.tf` を生成します。

## 対応技術スタック

| カテゴリ | 対応項目 |
|---------|---------|
| フロントエンド | React (Vite), Vue.js (Vite) |
| バックエンド | Node.js, Python (Flask), Go, PHP, Java |
| データベース | MySQL, PostgreSQL |
| キャッシュ | Redis |
| リバースプロキシ | Nginx（レートリミット、gzip、WebSocket対応） |
| AWS ローカル開発 | LocalStack (S3, SQS, SNS, DynamoDB) |
| 監視 | Prometheus, Grafana, Loki, Promtail |
| CI/CD | GitHub Actions（Trivy スキャン、golangci-lint 含む） |
| クラウド | AWS, GCP, Azure（Terraform） |
| オーケストレーション | Kubernetes |

## アーキテクチャ

```
dev-box/
├── cmd/                  # CLI コマンドハンドラ（薄いレイヤー）
├── internal/
│   ├── config/           # 設定構造体
│   ├── generator/        # 各種ファイル生成ロジック
│   ├── port/             # 空きポート探索
│   └── security/         # パスワード生成
├── main.go
├── Makefile
└── go.mod
```

### 設計方針

- **責務の分離**: `cmd/` は入出力のみ。ビジネスロジックは `internal/` に集約
- **テンプレートエンジン**: `text/template` による構造化された設定ファイル生成
- **エラーハンドリング**: すべての I/O 操作でエラーを検査・報告
- **セキュリティ**: `crypto/rand` による暗号学的に安全なパスワード生成、`.env` への機密情報分離

## テスト

```bash
make test
```

テストカバレッジ:
- `internal/security` — 82.1%
- `internal/port` — 70.0%
- `internal/generator` — 32.8%

## 開発

```bash
make build    # バイナリをビルド
make test     # テスト実行
make lint     # go vet による静的解析
make clean    # ビルド成果物を削除
```

## ライセンス

[MIT License](LICENSE)
