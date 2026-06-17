// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// LoadOrNotConfigured wraps LoadMultiAppConfig with the standard "not yet
// configured vs. couldn't read" disambiguation that every config-required
// command should use:
//
//   - file missing → workspace-aware NotConfiguredError (init / bind hint)
//   - parse error / permission error → real load failure with the original
//     cause preserved, so the user can actually fix the broken file
//
// Without this, every call site that did `if err != nil { return
// NotConfiguredError() }` silently coerced corrupt-config into "run init",
// which sent users in circles when their config.json was just malformed.
func LoadOrNotConfigured() (*MultiAppConfig, error) {
	multi, err := LoadMultiAppConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, NotConfiguredError()
		}
		// Surface the real cause (parse error, permission denied, etc.)
		// so the user can fix the broken file. Wrapping as ConfigError
		// keeps it on the standard structured-envelope path at the root
		// command's error sink.
		return nil, &ConfigError{
			Code:    3,
			Type:    "config",
			Message: fmt.Sprintf("failed to load config: %v", err),
		}
	}
	if multi == nil || len(multi.Apps) == 0 {
		return nil, NotConfiguredError()
	}
	return multi, nil
}

const (
	// localInitHintTTY is shown to a human at an interactive terminal, where the
	// blocking `config init --new` is simplest — the QR renders right there and
	// the human can scan it while the command waits.
	localInitHintTTY = "run `lark-cli config init --new` in the background. It blocks and outputs a verification URL — retrieve the URL and open it in a browser to complete setup."

	// localInitHintNonTTY is shown to AI agents / piped callers, where the
	// blocking flow would deadlock (the agent can't surface the URL until the
	// command returns, but it won't return until the user scans). Point them at
	// the non-blocking two-step flow instead.
	localInitHintNonTTY = "run `lark-cli config init --new --no-wait` to get a device_code and verification_url immediately, then run `lark-cli config init --device-code <code>` after setup is completed in the browser."

	// agentBindHint is the canonical "you're in an Agent workspace, see
	// the binding workflow" guidance. Always points at --help (never a
	// ready-to-run bind command) so the AI reads the confirmation
	// discipline (identity preset, user opt-in) before acting.
	agentBindHint = "read `lark-cli config bind --help`, then ask the user to confirm intent and identity preset (bot-only or user-default); only after both are confirmed, run `lark-cli config bind`"
)

// isStdinTerminal reports whether stdin is an interactive terminal. It is a var
// so tests can exercise both the TTY and non-TTY hint branches.
var isStdinTerminal = func() bool {
	// internal/core sits below the IOStreams layer (cmdutil imports core, not
	// the reverse), so there is no IOStreams to consult here. Probe stdin
	// directly — a read-only TTY check used solely to pick the right
	// not-configured hint.
	return term.IsTerminal(int(os.Stdin.Fd())) //nolint:forbidigo // no IOStreams in core; read-only TTY probe for hint selection
}

// localInitHint returns the workspace-local "run init" guidance, choosing the
// blocking single command for an interactive terminal (a human can see the QR)
// and the non-blocking two-step flow for AI agents / non-interactive callers
// (where blocking would deadlock).
func localInitHint() string {
	if isStdinTerminal() {
		return localInitHintTTY
	}
	return localInitHintNonTTY
}

// NotConfiguredError returns the canonical "not configured" error, with a
// hint that depends on the active workspace:
//
//   - WorkspaceLocal → suggest `config init --new` (creates a new app).
//   - WorkspaceOpenClaw / WorkspaceHermes → point at `config bind --help`
//     rather than a ready-to-run command, because binding is policy-laden:
//     the user must pick an identity preset (bot-only vs user-default),
//     and re-binding may overwrite an existing one. The help text walks
//     the AI through the confirmation flow.
//
// All "config not loaded yet" call sites should use this helper rather than
// hand-rolling a hint, so AI agents always get a workspace-correct next step.
func NotConfiguredError() error {
	ws := CurrentWorkspace()
	if ws.IsLocal() {
		return &ConfigError{
			Code:    3,
			Type:    "config",
			Message: "not configured",
			Hint:    localInitHint(),
		}
	}
	return &ConfigError{
		Code:    3,
		Type:    ws.Display(),
		Message: fmt.Sprintf("%s context detected but lark-cli is not bound to it", ws.Display()),
		Hint:    agentBindHint,
	}
}

// reconfigureHint returns the workspace-aware "fix it from scratch" hint
// used by error paths that aren't full ConfigErrors (e.g. plain fmt.Errorf
// strings from keychain / secret validation). Local → `config init`;
// Agent → `config bind --help` so the AI reads the binding workflow and
// confirms identity preset with the user before running the actual command.
func reconfigureHint() string {
	if CurrentWorkspace().IsLocal() {
		return "please run `lark-cli config init` to reconfigure"
	}
	return agentBindHint
}

// NoActiveProfileError mirrors NotConfiguredError for the related
// "config exists but the requested profile cannot be resolved" case. In agent
// workspaces a missing profile typically means the binding was wiped while
// the workspace marker remained — re-binding is the correct fix, not init.
func NoActiveProfileError() error {
	ws := CurrentWorkspace()
	if ws.IsLocal() {
		return &ConfigError{
			Code:    3,
			Type:    "config",
			Message: "no active profile",
			Hint:    localInitHint(),
		}
	}
	return &ConfigError{
		Code:    3,
		Type:    ws.Display(),
		Message: fmt.Sprintf("no active profile in %s workspace", ws.Display()),
		Hint:    agentBindHint,
	}
}
