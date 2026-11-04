# DevBox Studio

DevBox Studio は、`dev-box` CLI をブラウザから扱うための Web アプリケーションです。
フロントエンド、バックエンド、データベース、キャッシュ、Nginx、監視、CI などの構成を選ぶと、`dev-box` が生成するファイルを画面上でプレビューできます。

## 目的

`dev-box` は、開発環境のひな形を生成する Go 製 CLI です。
DevBox Studio は、その CLI を単体ツールとして終わらせず、ブラウザから使えるプロダクト体験に拡張するために作っています。

Go API はローカルの `dev-box init --dry-run --format json` を呼び出し、React UI は返ってきた Docker Compose、アプリケーションコード、運用設定、CI 設定などをファイル単位で表示します。

## 主な機能

- 技術スタックをブラウザ上で選択（クラウドを選ぶと Terraform も対象）
- `dev-box` が生成するファイルを事前プレビュー
- 生成ファイルの種類、パス、行数、内容を確認
- 選択した構成でプロジェクト一式を出力
- `dev-box` 本体の JSON 出力を利用した安定した連携

## ローカル開発

API を起動します。

```powershell
Set-Location "C:\Job hunting\dev-box\devbox-studio\backend"
go run .
```

別のターミナルでフロントエンドを起動します。

```powershell
Set-Location "C:\Job hunting\dev-box\devbox-studio\frontend"
npm.cmd install
npm.cmd run dev -- --host 127.0.0.1
```

ブラウザで次のURLを開きます。

```text
http://127.0.0.1:5173/
```

フロントエンドは `/api` へのリクエストを、`8080` 番ポートで動く Go API にプロキシします。

## 設定

| 環境変数 | 対象 | 既定値 | 内容 |
|---|---|---|---|
| `DEV_BOX_SOURCE_DIR` | API | リポジトリのルート | `dev-box` のソースディレクトリ |
| `DEV_BOX_OUTPUT_ROOT` | API | `devbox-studio/generated` | 「プロジェクトを生成」の出力先 |
| `BACKEND_HOST` | API | `127.0.0.1` | 待ち受けアドレス |
| `BACKEND_PORT` | API | `8080` | 待ち受けポート |
| `STUDIO_API_URL` | Vite | `http://127.0.0.1:8080` | `/api` のプロキシ先 |

`8080` 番が使用中の場合は、ポートを変えて両方を起動できます。

```powershell
$env:BACKEND_PORT = "18080"; go run .
$env:STUDIO_API_URL = "http://127.0.0.1:18080"; npm.cmd run dev -- --host 127.0.0.1
```

## セキュリティ

API はローカルの `dev-box` を実行してファイルを書き込むため、手元の端末以外から操作されないように次の制限をかけています。

- 既定で `127.0.0.1` のみで待ち受けます。コンテナなどで変更する場合だけ `BACKEND_HOST` を設定してください。
- `Host` ヘッダーと `Origin` ヘッダーがループバック（`localhost`、`127.0.0.1`、`::1`）でないリクエストは `403` で拒否します。閲覧中の他の Web サイトからの操作や DNS rebinding を防ぎます。
- `Access-Control-Allow-Origin` は返しません。フロントエンドは Vite のプロキシ経由の同一オリジンで通信します。
- POST は `Content-Type: application/json` のみ受け付け、本文は 64 KiB までです。
- プロジェクト名は `[a-zA-Z0-9][a-zA-Z0-9_-]{1,39}` に限定し、出力先は `generated/` の外に出られません。
- プレビューでは `.env` のパスワード・シークレット・トークン・キーを伏せます。

## 動作確認

```powershell
Set-Location "C:\Job hunting\dev-box"
go test ./...

Set-Location "C:\Job hunting\dev-box\devbox-studio\backend"
go test ./...

Set-Location "C:\Job hunting\dev-box\devbox-studio\frontend"
npm.cmd run build
```