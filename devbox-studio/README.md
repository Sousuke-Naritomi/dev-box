# DevBox Studio

DevBox Studio is a browser interface for the `dev-box` CLI. It lets a user choose
a web stack, preview the exact files `dev-box` will create, and generate a
project without leaving the browser.

## Why this exists

The original `dev-box` project is a Go CLI for generating development
environments. This application turns that CLI into a product experience: a Go API
wraps the local `dev-box` command, and a React UI presents the generated Compose,
application, operations, and CI files.

## Local development

Start the API:

```powershell
Set-Location backend
$env:DEV_BOX_SOURCE_DIR = "C:\Job hunting\dev-box"
go run .
```

Start the frontend in another terminal:

```powershell
Set-Location frontend
npm.cmd install
npm.cmd run dev
```

The frontend proxies `/api` requests to the Go API on port `8080`.
