# CA Bundle Extraction from OS Certificate Store

## Problem

The base module's `gitSSLCAInfoStep` points git at `~/.config/shhh/ca-bundle.pem`, but nothing creates that file. The current design assumes the user has a PEM file, which is wrong — corporate users won't have one. The certificates live in the OS certificate store, injected by corporate IT (e.g., TLS-intercepting proxy root CAs).

## Solution

Add a new `CertStore` platform interface and a "Build CA bundle" step in the base module that extracts trusted root CAs from the OS cert store, appends any configured extras, and writes a PEM bundle file.

## Architecture

### Platform Interface

```go
// platform/certstore.go
type CertStore interface {
    // SystemRoots returns all trusted root certificates from the OS cert store.
    SystemRoots() ([]*x509.Certificate, error)
}
```

### Platform Implementations

**Windows** (`certstore_windows.go`):
- Opens both `CurrentUser\Root` and `LocalMachine\Root` stores via `syscall.CertOpenSystemStore`
- Enumerates with `CertEnumCertificatesInStore`, parses each DER blob with `x509.ParseCertificate`
- Deduplicates by SHA-256 fingerprint of raw bytes (same CA can appear in both stores)
- No admin required — both stores are readable by any user

**macOS** (`certstore_darwin.go`):
- Runs `security find-certificate -a -p` against `/System/Library/Keychains/SystemRootCertificates.keychain` and `/Library/Keychains/System.keychain`
- Parses PEM output, decodes each block with `x509.ParseCertificate`
- Falls back gracefully if a keychain path doesn't exist
- Real extraction (not a stub) — useful for dev testing the full flow

**Linux** (`certstore_other.go`):
- Returns `ErrNotSupported` — same pattern as existing env/profile stubs

**Mock** (`mock/certstore.go`):
- Holds a `[]*x509.Certificate` slice injected at test time

### The "Build CA bundle" Step

New step inserted into `base.go` before `gitSSLCAInfoStep`. Step ordering becomes:

```
proxy env vars → build CA bundle → git ssl.caInfo → git default branch
```

**Check:** Computes a hash of current system roots (call `CertStore.SystemRoots()`, sort certs by raw DER bytes, concatenate with contents of each `certs.extra` file, SHA-256 the result). Compares against `state.CABundleHash`. If they match and the bundle file exists on disk, skip. This detects new certs added to the OS store or new extras added to the TOML config.

**Run:**
1. Call `deps.CertStore.SystemRoots()` to get all system root CAs
2. Encode each cert as PEM into a buffer
3. For each path in `config.Certs.Extra`, read the file, validate it contains valid PEM cert(s), append to the buffer
4. Write atomically to `config.CABundlePath()` (write temp file, rename)
5. Compute SHA-256 of the bundle, store in `state.CABundleHash`
6. Set `SSL_CERT_FILE` env var (in-process via `os.Setenv` + persistent via `deps.Env.Set`) so tools like pip, curl, etc. pick it up

**DryRun:** "Would extract N certs from system store and write to ~/.config/shhh/ca-bundle.pem"

**Explain:** "Corporate networks often use TLS-intercepting proxies with custom root certificates. Most dev tools (git, pip, npm, curl) need a PEM file with these certificates to verify HTTPS connections. We extract them from your OS certificate store and bundle them into a single file that all your tools can use."

### Dependencies Struct Change

```go
type Dependencies struct {
    Config    *config.Config
    Env       platform.UserEnv
    Profile   platform.ProfileManager
    CertStore platform.CertStore        // new
    Exec      shexec.Runner
    State     *state.State
}
```

## Edge Cases

- **Empty cert store:** Fail with clear error "No root certificates found in system store." Don't write an empty bundle.
- **Invalid extras:** Fail naming the bad file path. Don't silently skip — the org config specified it for a reason.
- **Bundle freshness:** Hash covers system roots + extras. Adding a new `certs.extra` entry or corporate IT pushing a new root CA triggers re-generation on next run.
- **File permissions:** `0644` — contains only public certificates.
- **Atomic write:** Write to temp file in same directory, then rename. Prevents half-written bundles.
- **Hash determinism:** Sort certs by raw DER bytes before hashing. OS enumeration order may vary.

## Config

Existing config fields (no changes needed):

```toml
[certs]
source = "system"          # extract from OS cert store
extra = ["/path/to/ca.pem"] # additional PEM files to append
```

`certs.extra` entries are local file paths to PEM-encoded certificates.

## Testing

**CertStore implementation tests** (platform-gated):
- `certstore_darwin_test.go`: Verify >0 certs returned, each `IsCA: true`, no duplicates
- `certstore_windows_test.go`: Same assertions, `//go:build windows`

**CA bundle step tests** (platform-independent, using mock):
- `Run` writes valid PEM containing all mock certs
- `Run` appends extras from temp PEM files
- `Check` returns true when hash matches and file exists
- `Check` returns false when new cert added to mock store
- `Run` fails on empty cert store
- `Run` fails on missing extra file path
- Bundle round-trips through `pem.Decode`

**Integration test update:**
- Add `mock.CertStore` to `setup.Dependencies` in existing `TestFullSetupFlow`
- Verify CA bundle step runs before git SSL step
- Verify `state.CABundleHash` populated after run

## Dependencies

No new external dependencies. Uses only stdlib: `crypto/x509`, `crypto/sha256`, `encoding/pem`, `syscall` (Windows).

## Files

| File | Action |
|------|--------|
| `internal/platform/certstore.go` | Create — interface definition |
| `internal/platform/certstore_windows.go` | Create — Windows CryptoAPI impl |
| `internal/platform/certstore_darwin.go` | Create — macOS security command impl |
| `internal/platform/certstore_other.go` | Create — Linux stub |
| `internal/platform/mock/certstore.go` | Create — test mock |
| `internal/module/setup/base.go` | Modify — add caBundleStep, update Dependencies |
| `internal/module/setup/base_test.go` | Modify — add bundle step tests |
| `internal/integration_test.go` | Modify — add mock CertStore to deps |
