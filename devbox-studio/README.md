# DevBox Studio

DevBox Studio は、`dev-box` CLI をブラウザから扱うための Web アプリケーションです。
フロントエンド、バックエンド、データベース、キャッシュ、Nginx、監視、CI などの構成を選ぶと、`dev-box` が生成するファイルを画面上でプレビューできます。

## 目的

`dev-box` は、開発環境のひな形を生成する Go 製 CLI です。
DevBox Studio は、その CLI を単体ツールとして終わらせず、ブラウザから使えるプロダクト体験に拡張するために作っています。

Go API はローカルの `dev-box init --dry-run --format json` を呼び出し、React UI は返ってきた Docker Compose、アプリケーションコード、運用設定、CI 設定などをファイル単位で表示します。

## 主な機能

- 技術スタックをブラウザ上で選択
- `dev-box` が生成するファイルを事前プレビュー
- 生成ファイルの種類、パス、行数、内容を確認
- 選択した構成でプロジェクト一式を出力
- `dev-box` 本体の JSON 出力を利用した安定した連携

## ローカル開発

API を起動します。

```powershell
Set-Location "C:\Job hunting\dev-box\devbox-studio\backend"
$env:DEV_BOX_SOURCE_DIR = "C:\Job hunting\dev-box"
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

## 動作確認

```powershell
Set-Location "C:\Job hunting\dev-box"
go test ./...

Set-Location "C:\Job hunting\dev-box\devbox-studio\backend"
go test ./...

Set-Location "C:\Job hunting\dev-box\devbox-studio\frontend"
npm.cmd run build
```