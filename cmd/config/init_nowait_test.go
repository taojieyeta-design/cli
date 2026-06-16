// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/larksuite/cli/errs"
	larkauth "github.com/larksuite/cli/internal/auth"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
)

// --- cache round-trip ---

func TestInitNoWaitCache_RoundTrip(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	rec := initNoWaitRecord{
		Version:      initNoWaitCacheVersion,
		Brand:        "feishu",
		ProfileName:  "work",
		Lang:         "zh_cn",
		LangExplicit: true,
		Interval:     5,
		ExpiresAt:    time.Now().Unix() + 300,
		ConfigDigest: "abc123",
	}
	const dc = "device-code-xyz"

	if err := saveInitNoWaitRecord(dc, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadInitNoWaitRecord(dc)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got == nil {
		t.Fatal("load returned nil for a saved record")
	}
	if *got != rec {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", *got, rec)
	}

	if err := removeInitNoWaitRecord(dc); err != nil {
		t.Fatalf("remove: %v", err)
	}
	got2, err := loadInitNoWaitRecord(dc)
	if err != nil {
		t.Fatalf("load after remove: %v", err)
	}
	if got2 != nil {
		t.Errorf("expected nil after remove, got %+v", got2)
	}
	// Removing a non-existent record must be a no-op, not an error.
	if err := removeInitNoWaitRecord(dc); err != nil {
		t.Errorf("remove of missing record should be nil, got %v", err)
	}
}

func TestInitNoWaitCache_LoadMissing(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	got, err := loadInitNoWaitRecord("never-saved")
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing record, got %+v", got)
	}
}

func TestInitNoWaitCache_VersionMismatchIgnored(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	const dc = "stale-version"
	rec := initNoWaitRecord{Version: initNoWaitCacheVersion + 1, ExpiresAt: time.Now().Unix() + 300}
	if err := saveInitNoWaitRecord(dc, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadInitNoWaitRecord(dc)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for version mismatch, got %+v", got)
	}
	// The stale entry should have been discarded by the load.
	got2, _ := loadInitNoWaitRecord(dc)
	if got2 != nil {
		t.Errorf("stale-version entry was not removed on load")
	}
}

func TestInitNoWaitCacheKey(t *testing.T) {
	// Distinct device codes that a char-replacement sanitizer would collide
	// ("a/b" and "a:b" -> "a_b") must map to distinct keys.
	if initNoWaitCacheKey("a/b") == initNoWaitCacheKey("a:b") {
		t.Error("distinct device codes must not collide on the cache key")
	}
	// Deterministic.
	if initNoWaitCacheKey("xyz") != initNoWaitCacheKey("xyz") {
		t.Error("cache key must be deterministic")
	}
	// sha256 hex: 64 chars, filesystem-safe regardless of input.
	k := initNoWaitCacheKey("has /, :, ;, spaces and 'quotes'")
	if len(k) != 64 {
		t.Errorf("expected 64-char sha256 hex key, got %d: %q", len(k), k)
	}
}

// --- config digest ---

func TestComputeConfigDigest(t *testing.T) {
	if d := computeConfigDigest(nil); d != "" {
		t.Errorf("nil digest = %q, want empty", d)
	}
	cfg1 := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "cli_a", Brand: core.BrandFeishu}}}
	cfg1Dup := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "cli_a", Brand: core.BrandFeishu}}}
	cfg2 := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "cli_b", Brand: core.BrandFeishu}}}

	if computeConfigDigest(cfg1) == "" {
		t.Error("non-nil config digest should be non-empty")
	}
	if computeConfigDigest(cfg1) != computeConfigDigest(cfg1Dup) {
		t.Error("equal configs should produce equal digests")
	}
	if computeConfigDigest(cfg1) == computeConfigDigest(cfg2) {
		t.Error("different configs should produce different digests")
	}
}

// --- failure classification for cache cleanup ---

func TestAppRegShouldClearCache(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"success", nil, true},
		{"denied", larkauth.ErrAppRegDenied, true},
		{"expired", larkauth.ErrAppRegExpired, true},
		{"expired wrapped", fmt.Errorf("%w, please try again", larkauth.ErrAppRegExpired), true},
		{"timeout", larkauth.ErrAppRegTimeout, true},
		{"timeout wrapped", fmt.Errorf("%w, please try again", larkauth.ErrAppRegTimeout), true},
		{"cancelled", larkauth.ErrAppRegCancelled, false},
		{"transient generic", fmt.Errorf("network boom"), false},
		{"missing fields", fmt.Errorf("app registration succeeded but missing client_id or client_secret"), false},
	}
	for _, c := range cases {
		if got := appRegShouldClearCache(c.err); got != c.want {
			t.Errorf("%s: appRegShouldClearCache = %v, want %v", c.name, got, c.want)
		}
	}
}

// --- flag validation (returns before any network) ---

func TestConfigInitRun_NoWaitAndDeviceCodeMutuallyExclusive(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), NoWait: true, DeviceCode: "x"}
	assertValidationParam(t, configInitRun(opts), "--device-code")
}

func TestConfigInitRun_NoWaitWithAppIDRejected(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), NoWait: true, AppID: "cli_x"}
	assertValidationParam(t, configInitRun(opts), "--no-wait")
}

// The conflict error must point at the flag the caller actually passed: with
// --device-code (not --no-wait) + --app-id, remediation should name --device-code.
func TestConfigInitRun_DeviceCodeWithAppIDReportsDeviceCode(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), DeviceCode: "dc", AppID: "cli_x"}
	assertValidationParam(t, configInitRun(opts), "--device-code")
}

// --- resume guards (return before any network) ---

func TestResumeAppRegistration_NoCacheEntry(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), DeviceCode: "missing-dc"}
	assertValidationParam(t, resumeAppRegistration(opts), "--device-code")
}

func TestResumeAppRegistration_ExpiredClearsCache(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	const dc = "expired-dc"
	rec := initNoWaitRecord{
		Version:   initNoWaitCacheVersion,
		Brand:     "feishu",
		Interval:  5,
		ExpiresAt: time.Now().Unix() - 10, // already past
	}
	if err := saveInitNoWaitRecord(dc, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), DeviceCode: dc}
	assertValidationParam(t, resumeAppRegistration(opts), "--device-code")

	if got, _ := loadInitNoWaitRecord(dc); got != nil {
		t.Error("expired cache entry should have been removed")
	}
}

// A cache file that exists but cannot be parsed is a storage failure, not a
// "no pending creation" validation error — the user should fix storage rather
// than assume the device code is bad.
func TestResumeAppRegistration_CorruptCacheIsStorageError(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	const dc = "corrupt-dc"
	if err := os.MkdirAll(initNoWaitCacheDir(), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(initNoWaitCachePath(dc), []byte("{ not valid json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), DeviceCode: dc}
	err := resumeAppRegistration(opts)
	var intErr *errs.InternalError
	if !errors.As(err, &intErr) {
		t.Fatalf("expected *errs.InternalError for unreadable cache, got %T: %v", err, err)
	}
}

func TestResumeAppRegistration_ConfigDrift(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	const dc = "drift-dc"
	rec := initNoWaitRecord{
		Version:      initNoWaitCacheVersion,
		Brand:        "feishu",
		Interval:     5,
		ExpiresAt:    time.Now().Unix() + 300,
		ConfigDigest: "stale-digest-that-will-not-match-current-config",
	}
	if err := saveInitNoWaitRecord(dc, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	opts := &ConfigInitOptions{Factory: f, Ctx: context.Background(), DeviceCode: dc}
	assertValidationParam(t, resumeAppRegistration(opts), "--device-code")
}
