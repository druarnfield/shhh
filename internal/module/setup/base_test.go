package setup

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/platform/mock"
	"github.com/druarnfield/shhh/internal/state"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("SSL_CERT_FILE")
	os.Unsetenv("GOPATH")
	os.Unsetenv("GOPROXY")
	os.Unsetenv("UV_PYTHON_PREFERENCE")
	os.Unsetenv("UV_INDEX_URL")
	os.Unsetenv("PIP_INDEX_URL")
	os.Unsetenv("REQUESTS_CA_BUNDLE")
	os.Unsetenv("PIP_CERT")
	os.Unsetenv("NODE_EXTRA_CA_CERTS")
	os.Exit(code)
}

func testDeps() *Dependencies {
	return &Dependencies{
		Config:    testConfig(),
		Env:       mock.NewUserEnv(),
		Profile:   mock.NewProfileManager("/tmp/test_profile.ps1"),
		CertStore: mock.NewCertStore(testCerts()),
		Exec:      &exec.MockRunner{Results: map[string]exec.Result{}},
		State:     &state.State{},
	}
}

// testCerts generates 2 self-signed CA certificates for testing.
func testCerts() []*x509.Certificate {
	certs := make([]*x509.Certificate, 2)
	for i := range certs {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic(err)
		}

		template := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 1)),
			Subject: pkix.Name{
				CommonName: "Test CA " + string(rune('A'+i)),
			},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().Add(24 * time.Hour),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
		}

		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			panic(err)
		}

		cert, err := x509.ParseCertificate(der)
		if err != nil {
			panic(err)
		}
		certs[i] = cert
	}
	return certs
}

func testConfig() *config.Config {
	cfg := config.Defaults()
	cfg.Proxy.HTTP = "http://proxy:8080"
	cfg.Proxy.HTTPS = "http://proxy:8080"
	cfg.Proxy.NoProxy = "localhost,127.0.0.1,.internal"
	cfg.Certs.Source = "system"
	cfg.Git.DefaultBranch = "main"
	cfg.Scoop.Buckets = []string{"extras", "versions"}
	cfg.Registries.GoProxy = "https://goproxy.example.com"
	cfg.Registries.PyPIMirror = "https://pypi.example.com/simple"
	cfg.Registries.NPMRegistry = "https://npm.example.com/"
	cfg.Tools.Core = []string{"git", "jq", "ripgrep"}
	cfg.Tools.Data = []string{"sqlcmd"}
	cfg.Tools.Optional = []string{"bat", "lazygit"}
	return cfg
}

func TestBaseModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)

	if mod.ID != "base" {
		t.Errorf("ID = %q, want %q", mod.ID, "base")
	}

	if len(mod.Steps) == 0 {
		t.Fatal("base module has no steps")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Set HTTP_PROXY", "Set HTTPS_PROXY", "Set NO_PROXY", "Build CA bundle", "Install Scoop", "Add Scoop buckets"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestProxySteps_SetEnvVars(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)
	ctx := context.Background()

	// Run proxy steps (first 3)
	for i := 0; i < 3 && i < len(mod.Steps); i++ {
		step := mod.Steps[i]
		if step.Check(ctx) {
			t.Errorf("step %q should not be already done", step.Name)
		}
		if err := step.Run(ctx); err != nil {
			t.Fatalf("step %q: %v", step.Name, err)
		}
	}

	// Verify env vars were set in mock
	val, _, err := deps.Env.Get("HTTP_PROXY")
	if err != nil {
		t.Fatalf("Get HTTP_PROXY: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("HTTP_PROXY = %q, want %q", val, "http://proxy:8080")
	}

	// Verify in-process env was set
	if got := os.Getenv("HTTP_PROXY"); got != "http://proxy:8080" {
		t.Errorf("os.Getenv(HTTP_PROXY) = %q, want %q", got, "http://proxy:8080")
	}
}

func TestProxySteps_CheckSkipsIfDone(t *testing.T) {
	deps := testDeps()
	deps.Env.Set("HTTP_PROXY", "http://proxy:8080")
	os.Setenv("HTTP_PROXY", "http://proxy:8080")
	defer os.Unsetenv("HTTP_PROXY")

	mod := NewBaseModule(deps)
	ctx := context.Background()

	step := mod.Steps[0] // HTTP_PROXY step
	if !step.Check(ctx) {
		t.Error("step should be marked as done")
	}
}

func TestProxySteps_DryRun(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)
	ctx := context.Background()

	for i := 0; i < 3 && i < len(mod.Steps); i++ {
		step := mod.Steps[i]
		if step.DryRun == nil {
			t.Errorf("step %q has no DryRun", step.Name)
			continue
		}
		msg := step.DryRun(ctx)
		if msg == "" {
			t.Errorf("step %q DryRun returned empty", step.Name)
		}
	}
}

func TestCABundleStep_Run_WritesPEM(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}

	step := caBundleStep(deps)
	ctx := context.Background()

	// The step writes to config.CABundlePath(), so ensure the dir exists.
	bundlePath := config.CABundlePath()
	os.MkdirAll(filepath.Dir(bundlePath), 0755)
	defer os.Remove(bundlePath)

	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Read the bundle and verify it's valid PEM with the expected certs.
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("reading bundle: %v", err)
	}

	var count int
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			t.Errorf("unexpected PEM type: %q", block.Type)
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			t.Errorf("invalid cert in bundle: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("bundle has %d certs, want 2", count)
	}

	// Verify hash was set.
	if deps.State.CABundleHash == "" {
		t.Error("CABundleHash not set after Run")
	}

	// Verify SSL_CERT_FILE was set.
	if got := os.Getenv("SSL_CERT_FILE"); got != bundlePath {
		t.Errorf("SSL_CERT_FILE = %q, want %q", got, bundlePath)
	}
}

func TestCABundleStep_Run_AppendsExtras(t *testing.T) {
	deps := testDeps()

	// Create a temp extra PEM file.
	extraDir := t.TempDir()
	extraPath := filepath.Join(extraDir, "extra-ca.pem")
	extraCerts := testCerts()[:1]
	extraPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: extraCerts[0].Raw,
	})
	if err := os.WriteFile(extraPath, extraPEM, 0644); err != nil {
		t.Fatalf("writing extra cert: %v", err)
	}

	deps.Config.Certs.Extra = []string{extraPath}

	step := caBundleStep(deps)
	ctx := context.Background()

	bundlePath := config.CABundlePath()
	os.MkdirAll(filepath.Dir(bundlePath), 0755)
	defer os.Remove(bundlePath)

	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("reading bundle: %v", err)
	}

	var count int
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		count++
	}
	// 2 from mock + 1 extra.
	if count != 3 {
		t.Errorf("bundle has %d certs, want 3 (2 system + 1 extra)", count)
	}
}

func TestCABundleStep_Check_TrueWhenHashMatches(t *testing.T) {
	deps := testDeps()

	step := caBundleStep(deps)
	ctx := context.Background()

	bundlePath := config.CABundlePath()
	os.MkdirAll(filepath.Dir(bundlePath), 0755)
	defer os.Remove(bundlePath)

	// Run to create the file and set the hash.
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Check should now return true.
	if !step.Check(ctx) {
		t.Error("Check returned false after Run, want true")
	}
}

func TestCABundleStep_Check_FalseWhenHashDiffers(t *testing.T) {
	deps := testDeps()

	step := caBundleStep(deps)
	ctx := context.Background()

	bundlePath := config.CABundlePath()
	os.MkdirAll(filepath.Dir(bundlePath), 0755)
	defer os.Remove(bundlePath)

	// Run to create the file and set the hash.
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Simulate a new cert being added by changing the mock certs.
	newCerts := append(testCerts(), testCerts()[0])
	deps.CertStore = mock.NewCertStore(newCerts)

	// Recreate step with updated deps to pick up new CertStore.
	step = caBundleStep(deps)

	if step.Check(ctx) {
		t.Error("Check returned true after cert change, want false")
	}
}

func TestCABundleStep_Run_FailsOnEmptyCertStore(t *testing.T) {
	deps := testDeps()
	deps.CertStore = mock.NewCertStore(nil) // empty

	step := caBundleStep(deps)
	ctx := context.Background()

	if err := step.Run(ctx); err == nil {
		t.Error("Run should fail with empty cert store")
	}
}

func TestCABundleStep_Run_FailsOnMissingExtraFile(t *testing.T) {
	deps := testDeps()
	deps.Config.Certs.Extra = []string{"/nonexistent/extra-ca.pem"}

	step := caBundleStep(deps)
	ctx := context.Background()

	bundlePath := config.CABundlePath()
	os.MkdirAll(filepath.Dir(bundlePath), 0755)
	defer os.Remove(bundlePath)

	if err := step.Run(ctx); err == nil {
		t.Error("Run should fail with missing extra file")
	}
}

func TestInstallScoopStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()

	// Check returns false when scoop not found.
	step := installScoopStep(deps)
	if step.Check(ctx) {
		t.Error("Check should return false when scoop is not installed")
	}

	// Check returns true when scoop is found.
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop --version"] = exec.Result{Stdout: "v0.4.1\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when scoop is installed")
	}
}

func TestInstallScoopStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["powershell -NoProfile -Command Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; irm get.scoop.sh | iex"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installScoopStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(deps.State.ManagedPathEntries) == 0 {
		t.Error("expected scoop shims dir in managed path entries")
	}
}

func TestInstallScoopStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installScoopStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestScoopBucketsStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()

	// Check returns false when bucket list fails.
	step := scoopBucketsStep(deps)
	if step.Check(ctx) {
		t.Error("Check should return false when scoop bucket list fails")
	}

	// Check returns false when not all buckets present.
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop bucket list"] = exec.Result{Stdout: "extras\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false when not all buckets are present")
	}

	// Check returns true when all buckets present.
	mockExec.Results["scoop bucket list"] = exec.Result{Stdout: "extras\nversions\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when all buckets are present")
	}
}

func TestScoopBucketsStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop bucket list"] = exec.Result{Stdout: "extras\n", ExitCode: 0}
	mockExec.Results["scoop bucket add versions"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := scoopBucketsStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify only missing bucket was added.
	found := false
	for _, call := range mockExec.Calls {
		if call == "scoop bucket add versions" {
			found = true
		}
		if call == "scoop bucket add extras" {
			t.Error("should not add already-present bucket 'extras'")
		}
	}
	if !found {
		t.Error("expected call to add 'versions' bucket")
	}
}

func TestScoopBucketsStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := scoopBucketsStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestBaseModule_NoBucketsStep_WhenEmpty(t *testing.T) {
	deps := testDeps()
	deps.Config.Scoop.Buckets = nil
	mod := NewBaseModule(deps)

	for _, s := range mod.Steps {
		if s.Name == "Add Scoop buckets" {
			t.Error("Add Scoop buckets step should be omitted when config has no buckets")
		}
	}
}
