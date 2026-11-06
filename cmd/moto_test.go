package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

func TestInitMotoWritesTheServiceAndRecordsItInTheProjectFile(t *testing.T) {
	cfg := fullStack()
	cfg.Moto = true
	files, _, err := buildInitFiles(cfg, &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["docker-compose.yml"].Content, "motoserver/moto:"+config.VersionMoto) {
		t.Error("the compose file should run Moto")
	}
	if !strings.Contains(files["dev-box.yaml"].Content, "moto: true\n") {
		t.Errorf("dev-box.yaml must record moto so the project can be regenerated:\n%s", files["dev-box.yaml"].Content)
	}
}

func TestInitJSONReportsMotoAndNotLocalStack(t *testing.T) {
	cfg := fullStack()
	cfg.Moto = true
	var out bytes.Buffer
	if err := runInitLogic(context.Background(), cfg, "p", true, "json", &out, fs.NewMockFileSystem(), &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Config["moto"] != true {
		t.Errorf("config.moto = %v", got.Config["moto"])
	}
	if _, ok := got.Config["localstack"]; ok {
		t.Error("the localstack key was removed")
	}
}

func TestInitRejectsTheRemovedLocalStackFlag(t *testing.T) {
	resetFlags(t, initCmd.Flags())
	app.FS = fs.NewMockFileSystem()
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"init", "--frontend=none", "--backend=go", "--database=none", "--cache=none", "--localstack", "--force"})
	err := rootCmd.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "localstack") {
		t.Fatalf("--localstack no longer exists and should be reported as an unknown flag, got %v", err)
	}
}
