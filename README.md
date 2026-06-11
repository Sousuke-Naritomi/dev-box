# 📦 dev-box

こんにちは！インフラエンジニア・SREを目指して制作したポートフォリオ作品「**dev-box**」です。

開発チームでよくある「あれ？私のPCでは動くんですけどね…」という環境構築のトラブル。あれ、すごくもったいない時間ですよね。
このツールは、そんな「ローカルインフラの構築から本番のKubernetesデプロイ、そして運用監視までの悩み」を、Go言語のCLIツールでサクッと自動化・標準化するために作りました。

## ✨ なにができるの？

- **ターミナル上でポチポチ選ぶだけ（対話型UI）**
  `charmbracelet/huh` を使ったリッチで直感的なターミナルUIを採用しています。フロントエンド、バックエンド、データベース、キャッシュなどを選ぶだけで、一瞬で最適な `docker-compose.yml` と各言語のソースコード雛形が出来上がります。
- **インフラのプロ水準の自動構成（Nginx & LocalStack）**
  通信を振り分けるAPIゲートウェイとして Nginx を自動構築。さらに、AWSにお金を払わず手元で検証できる LocalStack にも対応しています。
- **「ポートが被って起動しない問題」を自動回避**
  起動前に空きポートを自動で探し、「3306がダメなら3307で立ち上げる」という開発者体験(DX)を高める機能を搭載しています。
- **Kubernetes (K8s) と Terraformへの完全対応**
  `dev-box deploy` でクラウドインフラ構築用の `main.tf` を生成し、さらに `dev-box k8s` コマンドで、本番運用レベルの Kubernetes マニフェスト（Deployment / Service）を一発で生成します。
- **DevSecOpsと高度なオブザーバビリティ**
  脆弱性スキャン(Trivy)を組み込んだ GitHub Actions のCI/CDパイプラインを自動生成。さらに Prometheus / Grafana / Loki / Promtail を統合した「メトリクスとログ」の両方を分析できる最強の監視運用基盤も一瞬で立ち上がります。

## 🛠️ 対応している技術スタック

今のところ、以下の幅広い技術をフルスタックでカバーしています。

- **フロントエンド:** React (Vite), Vue.js (Vite)
- **バックエンド:** Go言語, Node.js (Express), Python (FastAPI), PHP (Apache), Java (Spring Boot)
- **データベース:** PostgreSQL, MySQL
- **キャッシュ:** Redis
- **インフラ・ゲートウェイ:** Docker, Nginx (リバースプロキシ), LocalStack (AWSシミュレーター)
- **コンテナオーケストレーション:** Kubernetes
- **クラウド展開:** Terraform (AWS/GCP/Azure)
- **監視・ログ分析:** Prometheus, Grafana, Loki, Promtail
- **CI/CD・DevSecOps:** GitHub Actions, Trivy (コンテナ脆弱性診断)

## 🚀 使い方

### 1. 開発環境のセットアップ
まずはプロジェクトのフォルダでこのコマンドを叩き、欲しい構成や監視ツールの有無を選んでください。
```bash
dev-box init
```

### 2. 環境の起動・停止
Dockerの長ったらしいコマンドを打つ必要はありません。
```bash
dev-box start
dev-box stop
```

### 3. 本番環境・クラウドへのデプロイ準備
KubernetesマニフェストやTerraformのコードを生成して、IaCの第一歩を踏み出せます。
```bash
dev-box k8s
dev-box deploy
```
