package internal

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/logging"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/module/setup"
	"github.com/druarnfield/shhh/internal/platform/mock"
	"github.com/druarnfield/shhh/internal/state"
)

func integrationTestCerts() []*x509.Certificate {
	certs := make([]*x509.Certificate, 2)
	for i := range certs {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic(err)
		}
		template := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 1)),
			Subject:      pkix.Name{CommonName: "Integration Test CA " + string(rune('A'+i))},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
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

func TestFullSetupFlow(t *testing.T) {
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go")
	gobin := filepath.Join(home, "go", "bin")

	t.Cleanup(func() {
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
		os.Remove(config.CABundlePath())
	})

	// Config
	cfg := config.Defaults()
	cfg.Org.Name = "Test Org"
	cfg.Proxy.HTTP = "http://proxy:8080"
	cfg.Proxy.HTTPS = "http://proxy:8080"
	cfg.Proxy.NoProxy = "localhost,127.0.0.1"
	cfg.Git.DefaultBranch = "main"
	cfg.Scoop.Buckets = []string{"extras", "versions"}
	cfg.Golang.Version = "1.23"
	cfg.Registries.GoProxy = "https://goproxy.example.com"
	cfg.Python.Version = "3.12"
	cfg.Registries.PyPIMirror = "https://pypi.example.com/simple"
	cfg.Node.Version = "22"
	cfg.Registries.NPMRegistry = "https://npm.example.com/"
	cfg.Tools.Core = []string{"git", "jq"}
	cfg.Tools.Data = []string{"sqlcmd"}
	cfg.Tools.Optional = []string{"bat"}

	// Mocks
	testCerts := integrationTestCerts()
	env := mock.NewUserEnv()
	prof := mock.NewProfileManager("/tmp/test_profile.ps1")
	certStore := mock.NewCertStore(testCerts)
	mockExec := &exec.MockRunner{
		Results: map[string]exec.Result{
			// Base module
			"git config --global init.defaultBranch":                        {Stdout: "", ExitCode: 1},
			"git config --global init.defaultBranch main":                   {Stdout: "", ExitCode: 0},
			"git config --global http.sslCAInfo":                            {Stdout: "", ExitCode: 1},
			"git config --global http.sslCAInfo " + config.CABundlePath():   {Stdout: "", ExitCode: 0},
			"scoop --version":                                               {Stdout: "", ExitCode: 1},
			"powershell -NoProfile -Command Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; irm get.scoop.sh | iex": {ExitCode: 0},
			"scoop bucket list":                {Stdout: "", ExitCode: 0},
			"scoop bucket add extras":          {ExitCode: 0},
			"scoop bucket add versions":        {ExitCode: 0},
			// Go module
			"go version":                       {Stdout: "", ExitCode: 1},
			"scoop install go":                 {ExitCode: 0},
			"go env GOPROXY":                   {Stdout: "", ExitCode: 1},
			"go env -w GOPROXY=https://goproxy.example.com": {ExitCode: 0},
			// Python module
			"uv --version":                     {Stdout: "", ExitCode: 1},
			"scoop install uv":                 {ExitCode: 0},
			"uv python list --only-installed":  {Stdout: "", ExitCode: 1},
			"uv python install 3.12":           {ExitCode: 0},
			// Node module
			"fnm --version":                    {Stdout: "", ExitCode: 1},
			"scoop install fnm":                {ExitCode: 0},
			"fnm list":                         {Stdout: "", ExitCode: 1},
			"fnm install 22":                   {ExitCode: 0},
			"fnm default 22":                   {ExitCode: 0},
			"fnm exec --using 22 -- npm config get cafile":                             {Stdout: "", ExitCode: 1},
			"fnm exec --using 22 -- npm config set cafile " + config.CABundlePath():    {ExitCode: 0},
			"fnm exec --using 22 -- npm config get registry":                           {Stdout: "", ExitCode: 1},
			"fnm exec --using 22 -- npm config set registry https://npm.example.com/":  {ExitCode: 0},
			// Tools module
			"scoop list":                       {Stdout: "", ExitCode: 0},
			"scoop install git":                {ExitCode: 0},
			"scoop install jq":                 {ExitCode: 0},
			"scoop install sqlcmd":             {ExitCode: 0},
			"scoop install bat":                {ExitCode: 0},
		},
	}
	st := &state.State{}

	deps := &setup.Dependencies{
		Config:    cfg,
		Env:       env,
		Profile:   prof,
		CertStore: certStore,
		Exec:      mockExec,
		State:     st,
	}

	// Registry — all modules
	reg := module.NewRegistry()
	reg.Register(setup.NewBaseModule(deps))
	reg.Register(setup.NewGolangModule(deps))
	reg.Register(setup.NewPythonModule(deps))
	reg.Register(setup.NewNodeModule(deps))
	reg.Register(setup.NewToolsModule(deps))

	// Run all modules
	logger := slog.New(logging.NopHandler{})
	runner := module.NewRunner(logger, false)

	allIDs := []string{"base", "golang", "python", "node", "tools"}
	results, err := runner.RunModules(context.Background(), reg, allIDs)
	if err != nil {
		t.Fatalf("RunModules: %v", err)
	}

	// Verify all modules completed successfully.
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("module %q error: %v", r.ModuleID, r.Err)
		}
		if r.Completed == 0 {
			t.Errorf("module %q: no steps completed", r.ModuleID)
		}
	}

	// Verify env vars set in mock.
	val, _, err := env.Get("HTTP_PROXY")
	if err != nil {
		t.Errorf("HTTP_PROXY not set in mock env: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("HTTP_PROXY = %q", val)
	}

	// Verify in-process env.
	if got := os.Getenv("HTTP_PROXY"); got != "http://proxy:8080" {
		t.Errorf("os HTTP_PROXY = %q", got)
	}
	if got := os.Getenv("GOPATH"); got != gopath {
		t.Errorf("os GOPATH = %q, want %q", got, gopath)
	}
	if got := os.Getenv("GOPROXY"); got != "https://goproxy.example.com" {
		t.Errorf("os GOPROXY = %q", got)
	}
	if got := os.Getenv("UV_PYTHON_PREFERENCE"); got != "only-managed" {
		t.Errorf("os UV_PYTHON_PREFERENCE = %q", got)
	}
	if got := os.Getenv("UV_INDEX_URL"); got != "https://pypi.example.com/simple" {
		t.Errorf("os UV_INDEX_URL = %q", got)
	}
	if got := os.Getenv("PIP_INDEX_URL"); got != "https://pypi.example.com/simple" {
		t.Errorf("os PIP_INDEX_URL = %q", got)
	}
	if got := os.Getenv("REQUESTS_CA_BUNDLE"); got != config.CABundlePath() {
		t.Errorf("os REQUESTS_CA_BUNDLE = %q, want %q", got, config.CABundlePath())
	}
	if got := os.Getenv("PIP_CERT"); got != config.CABundlePath() {
		t.Errorf("os PIP_CERT = %q, want %q", got, config.CABundlePath())
	}
	if got := os.Getenv("NODE_EXTRA_CA_CERTS"); got != config.CABundlePath() {
		t.Errorf("os NODE_EXTRA_CA_CERTS = %q, want %q", got, config.CABundlePath())
	}

	// Verify state updated.
	if len(st.ManagedEnvVars) == 0 {
		t.Error("state has no managed env vars")
	}
	if st.CABundleHash == "" {
		t.Error("CABundleHash not set after run")
	}
	if len(st.ScoopPackages) == 0 {
		t.Error("state has no scoop packages")
	}
	if len(st.ManagedPathEntries) == 0 {
		t.Error("state has no managed path entries")
	}

	// Verify GOBIN path was added.
	pathEntries, _ := env.ListPath()
	foundGOBIN := false
	for _, e := range pathEntries {
		if e.Dir == gobin {
			foundGOBIN = true
		}
	}
	if !foundGOBIN {
		t.Error("GOBIN not found in PATH entries")
	}

	// Verify profile has fnm init.
	block, _ := prof.ManagedBlock()
	if block == "" {
		t.Error("profile managed block should not be empty")
	}

	// Run again — should skip all steps (idempotency).
	mockExec2 := &exec.MockRunner{
		Results: map[string]exec.Result{
			// Base: all already configured.
			"scoop --version":                   {Stdout: "v0.4.1\n", ExitCode: 0},
			"git --version":                     {Stdout: "git version 2.47.0.windows.2\n", ExitCode: 0},
			"scoop bucket list":                 {Stdout: "extras\nversions\n", ExitCode: 0},
			"git config --global init.defaultBranch": {Stdout: "main\n", ExitCode: 0},
			"git config --global http.sslCAInfo":     {Stdout: config.CABundlePath() + "\n", ExitCode: 0},
			// Go: already installed.
			"go version":                        {Stdout: "go version go1.23.0 windows/amd64\n", ExitCode: 0},
			"go env GOPROXY":                    {Stdout: "https://goproxy.example.com\n", ExitCode: 0},
			// Python: already installed.
			"uv --version":                      {Stdout: "uv 0.4.0\n", ExitCode: 0},
			"uv python list --only-installed":   {Stdout: "cpython-3.12.0\n", ExitCode: 0},
			// Node: already installed.
			"fnm --version":                     {Stdout: "fnm 1.37.0\n", ExitCode: 0},
			"fnm list":                          {Stdout: "* v22.0.0\n", ExitCode: 0},
			"fnm exec --using 22 -- npm config get cafile":   {Stdout: config.CABundlePath() + "\n", ExitCode: 0},
			"fnm exec --using 22 -- npm config get registry": {Stdout: "https://npm.example.com/\n", ExitCode: 0},
			// Tools: already installed.
			"scoop list":                        {Stdout: "git\njq\nsqlcmd\nbat\n", ExitCode: 0},
		},
	}
	deps2 := &setup.Dependencies{
		Config:    cfg,
		Env:       env,
		Profile:   prof,
		CertStore: certStore,
		Exec:      mockExec2,
		State:     st,
	}
	reg2 := module.NewRegistry()
	reg2.Register(setup.NewBaseModule(deps2))
	reg2.Register(setup.NewGolangModule(deps2))
	reg2.Register(setup.NewPythonModule(deps2))
	reg2.Register(setup.NewNodeModule(deps2))
	reg2.Register(setup.NewToolsModule(deps2))

	runner2 := module.NewRunner(logger, false)
	results2, err := runner2.RunModules(context.Background(), reg2, allIDs)
	if err != nil {
		t.Fatalf("second RunModules: %v", err)
	}
	for _, r := range results2 {
		if r.Completed != 0 {
			t.Errorf("second run: module %q completed %d steps, want 0", r.ModuleID, r.Completed)
		}
		if r.Skipped == 0 {
			t.Errorf("second run: module %q skipped 0 steps, want >0", r.ModuleID)
		}
	}

	// Verify state round-trip.
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := state.Save(statePath, st); err != nil {
		t.Fatalf("Save state: %v", err)
	}
	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	if len(loaded.ManagedEnvVars) != len(st.ManagedEnvVars) {
		t.Error("state round-trip mismatch")
	}
}
