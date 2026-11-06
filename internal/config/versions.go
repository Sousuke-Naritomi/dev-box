package config

// Default component versions — pinned for reproducibility.
const (
	VersionNode     = "24-alpine"
	VersionPython   = "3.12-slim"
	VersionGo       = "1.27-alpine"
	VersionPHP      = "8.3-apache"
	VersionJava     = "17-jdk"
	VersionMySQL    = "8.4"
	VersionPostgres = "16-alpine"
	VersionRedis    = "7-alpine"
	VersionNginx    = "1.30-alpine"
	// Newer LocalStack images refuse to start without LOCALSTACK_AUTH_TOKEN (checked
	// 2026-11-06), and the open source repository is archived. Confirm a candidate
	// starts without credentials before bumping this.
	VersionLocalStack = "3.5"
	VersionPrometheus = "v3.16.0"
	VersionGrafana    = "13.3.0"
	VersionLoki       = "3.7.9"
	VersionAlloy      = "v1.21.3"

	// VersionTrivy is installed by the generated CI and by this repository's own
	// CI. It matches the default of aquasecurity/trivy-action@v0.36.0.
	VersionTrivy = "0.70.0"
)
