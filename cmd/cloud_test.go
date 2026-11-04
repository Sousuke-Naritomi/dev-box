package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

func typicalStack(cloud config.CloudProvider) config.ProjectConfig {
	cfg := fullStack()
	cfg.Cloud = cloud
	return cfg
}

func TestBuildInitFilesAddsTerraformOnlyWhenACloudIsSelected(t *testing.T) {
	gen := &generator.DefaultGenerator{}

	without, _, err := buildInitFiles(typicalStack(config.CloudNone), gen)
	if err != nil {
		t.Fatal(err)
	}
	for path := range without {
		if strings.HasPrefix(path, "terraform") {
			t.Errorf("no Terraform expected without a cloud, found %s", path)
		}
	}

	with, _, err := buildInitFiles(typicalStack(config.CloudAWS), gen)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.tf", "variables.tf", "terraform.tfvars.example", ".gitignore"} {
		if _, ok := with[filepath.Join("terraform", name)]; !ok {
			t.Errorf("expected terraform/%s", name)
		}
	}
	if _, ok := with["dev-box.yaml"]; !ok || !strings.Contains(with["dev-box.yaml"].Content, "cloud: aws\n") {
		t.Errorf("dev-box.yaml must record the cloud so the project can be regenerated:\n%s", with["dev-box.yaml"].Content)
	}
}

func TestInitTerraformIsSizedForTheSelectedStack(t *testing.T) {
	files, _, err := buildInitFiles(typicalStack(config.CloudAWS), &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	vars := files[filepath.Join("terraform", "variables.tf")].Content
	if !strings.Contains(vars, `default     = "t3.small"`) {
		t.Errorf("react+go+postgres+redis should default to t3.small:\n%s", vars)
	}
	if !strings.Contains(files[filepath.Join("terraform", "main.tf")].Content, `"react-go-postgres-redis"`) {
		t.Error("the instance should be tagged with the stack")
	}
}

func TestInitRejectsUnknownCloudFlag(t *testing.T) {
	resetFlags(t, initCmd.Flags())
	app.FS = fs.NewMockFileSystem()
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"init", "--frontend=none", "--backend=go", "--database=none", "--cache=none", "--cloud=heroku", "--force"})

	err := rootCmd.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cloud provider") {
		t.Fatalf("expected an unsupported cloud provider error, got %v", err)
	}
	if len(app.FS.(*fs.MockFileSystem).Files) != 0 {
		t.Error("nothing may be written for an invalid cloud")
	}
}

func TestInitCommandWithCloudFlagWritesTerraformDirectory(t *testing.T) {
	resetFlags(t, initCmd.Flags())
	mockFS := fs.NewMockFileSystem()
	app.FS = mockFS
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"init", "--frontend=none", "--backend=go", "--database=none", "--cache=none", "--cloud=gcp", "--force", "--output-dir=proj"})

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("init failed: %v\n%s", err, errOut.String())
	}
	for _, want := range []string{
		filepath.Join("proj", "terraform", "main.tf"),
		filepath.Join("proj", "terraform", ".gitignore"),
		filepath.Join("proj", "dev-box.yaml"),
	} {
		if !mockFS.Exists(want) {
			t.Errorf("expected %s", want)
		}
	}
	if !strings.Contains(out.String(), "terraform.tfvars.example") {
		t.Errorf("users should be told what to do next:\n%s", out.String())
	}
}

func TestInitJSONDryRunReportsCloudAndTerraformFiles(t *testing.T) {
	var out bytes.Buffer
	err := runInitLogic(context.Background(), typicalStack(config.CloudAzure), "studio-preview", true, "json", &out, fs.NewMockFileSystem(), &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	var got initOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Config.Cloud != config.CloudAzure {
		t.Errorf("config.cloud = %q", got.Config.Cloud)
	}
	found := false
	for _, f := range got.Files {
		if f.Path == "studio-preview/terraform/main.tf" {
			found = true
		}
	}
	if !found {
		t.Error("Studio relies on the JSON preview listing terraform/main.tf")
	}

	var none bytes.Buffer
	if err := runInitLogic(context.Background(), fullStack(), "x", true, "json", &none, fs.NewMockFileSystem(), &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(none.String(), `"cloud": "none"`) {
		t.Errorf("an unset cloud must be reported as none:\n%s", none.String())
	}
}

func TestInitMergesExistingTerraformGitignore(t *testing.T) {
	mockFS := fs.NewMockFileSystem()
	existing := filepath.Join("terraform", ".gitignore")
	mockFS.Files[existing] = fs.MockFile{Data: []byte("my-notes.txt\n*.tfstate\n"), Perm: 0o644}

	files, _, err := buildInitFiles(typicalStack(config.CloudAWS), &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedFiles(context.Background(), mockFS, ".", files, true, false, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := string(mockFS.Files[existing].Data)
	for _, want := range []string{"my-notes.txt", ".terraform/", "terraform.tfvars"} {
		if !strings.Contains(got, want) {
			t.Errorf("merged .gitignore should keep/add %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "*.tfstate\n") != 1 {
		t.Errorf("duplicate entries must not be added:\n%s", got)
	}
}

func TestDeployUsesTheCloudAndStackFromConfig(t *testing.T) {
	skipIfStdinIsTerminal(t)
	resetFlags(t, deployCmd.Flags())

	mockFS := fs.NewMockFileSystem()
	rendered, err := config.RenderProjectFile(typicalStack(config.CloudGCP))
	if err != nil {
		t.Fatal(err)
	}
	mockFS.Files["dev-box.yaml"] = fs.MockFile{Data: []byte(rendered), Perm: 0o644}
	app.FS = mockFS
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"deploy", "--config=dev-box.yaml", "--force", "--output-dir=tf"})

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("deploy failed: %v\n%s", err, errOut.String())
	}
	main := string(mockFS.Files[filepath.Join("tf", "main.tf")].Data)
	if !strings.Contains(main, "hashicorp/google") {
		t.Errorf("the provider should come from cloud: gcp in the config:\n%s", main)
	}
	if !strings.Contains(string(mockFS.Files[filepath.Join("tf", "variables.tf")].Data), `default     = "e2-small"`) {
		t.Error("the machine type should follow the stack in the config")
	}
}

func TestDeployFlagOverridesConfiguredCloud(t *testing.T) {
	skipIfStdinIsTerminal(t)
	resetFlags(t, deployCmd.Flags())

	mockFS := fs.NewMockFileSystem()
	rendered, err := config.RenderProjectFile(typicalStack(config.CloudGCP))
	if err != nil {
		t.Fatal(err)
	}
	mockFS.Files["dev-box.yaml"] = fs.MockFile{Data: []byte(rendered), Perm: 0o644}
	app.FS = mockFS
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"deploy", "--config=dev-box.yaml", "--provider=aws", "--force", "--output-dir=tf"})

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("deploy failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(string(mockFS.Files[filepath.Join("tf", "main.tf")].Data), "hashicorp/aws") {
		t.Error("--provider must win over the configured cloud")
	}
}

func TestDeployWithoutAnyProviderFailsClearlyWhenNonInteractive(t *testing.T) {
	skipIfStdinIsTerminal(t)
	resetFlags(t, deployCmd.Flags())

	mockFS := fs.NewMockFileSystem()
	rendered, err := config.RenderProjectFile(typicalStack(config.CloudNone))
	if err != nil {
		t.Fatal(err)
	}
	mockFS.Files["dev-box.yaml"] = fs.MockFile{Data: []byte(rendered), Perm: 0o644}
	app.FS = mockFS
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	for _, args := range [][]string{
		{"deploy"},
		{"deploy", "--config=dev-box.yaml"},
	} {
		rootCmd.SetArgs(args)
		err := rootCmd.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "--provider") {
			t.Errorf("%v: expected an error that points at --provider, got %v", args, err)
		}
	}
	for path := range mockFS.Files {
		if strings.HasPrefix(path, "terraform") {
			t.Errorf("nothing may be written without a provider: %s", path)
		}
	}
}

func TestInitDryRunDoesNotWriteTerraform(t *testing.T) {
	mockFS := fs.NewMockFileSystem()
	var out bytes.Buffer
	err := runInitLogic(context.Background(), typicalStack(config.CloudAWS), "p", true, "text", &out, mockFS, &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	if len(mockFS.Files) != 0 {
		t.Fatalf("dry run wrote files: %v", mockFS.Files)
	}
	if !strings.Contains(out.String(), filepath.Join("p", "terraform", "main.tf")) {
		t.Error("the dry run should show the Terraform files it would write")
	}
}
