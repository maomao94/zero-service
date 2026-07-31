/* global process */
/**
 * Trellis Dynamic Spec Injection Plugin
 *
 * OpenCode tool hooks cannot add context to the current model turn. A FULL
 * spec emission therefore blocks the write once with a model-visible error;
 * the shared Python engine records the emission, so the model's retry proceeds.
 */

import { execFileSync } from "child_process";
import { existsSync } from "fs";
import { join } from "path";
import { debugLog } from "../lib/trellis-context.js";

const PYTHON_CMD = process.platform === "win32" ? "python" : "python3";

function runSpecHook(directory, payload) {
  const hookPath = join(
    directory,
    ".opencode",
    "hooks",
    "inject-spec-context.py",
  );
  if (!existsSync(hookPath)) return null;

  try {
    const stdout = execFileSync(PYTHON_CMD, [hookPath], {
      cwd: directory,
      encoding: "utf-8",
      input: JSON.stringify(payload),
      stdio: ["pipe", "pipe", "pipe"],
      timeout: 15000,
    });
    if (!stdout.trim()) return null;
    const parsed = JSON.parse(stdout);
    return parsed?.hookSpecificOutput ?? null;
  } catch (error) {
    debugLog(
      "spec",
      "Spec hook failed open:",
      error instanceof Error ? error.message : String(error),
    );
    return null;
  }
}

function buildToolPayload(directory, input, output) {
  const tool = input?.tool?.toLowerCase();
  const args = output?.args;
  if (!args) return null;

  if (tool === "edit" || tool === "write") {
    if (typeof args.filePath !== "string" || !args.filePath.trim()) return null;
    return {
      hook_event_name: "PreToolUse",
      session_id: input?.sessionID,
      cwd: directory,
      tool_name: tool === "edit" ? "Edit" : "Write",
      tool_input: { file_path: args.filePath },
    };
  }

  if (tool === "apply_patch") {
    if (typeof args.patchText !== "string" || !args.patchText.trim())
      return null;
    return {
      hook_event_name: "PreToolUse",
      session_id: input?.sessionID,
      cwd: directory,
      tool_name: "apply_patch",
      tool_input: { command: args.patchText },
    };
  }

  return null;
}

export default async ({ directory }) => ({
  event: async ({ event }) => {
    if (
      process.env.TRELLIS_HOOKS === "0" ||
      process.env.TRELLIS_DISABLE_HOOKS === "1"
    ) {
      return;
    }
    if (event?.type !== "session.compacted" || !event?.properties?.sessionID) {
      return;
    }

    runSpecHook(directory, {
      hook_event_name: "SessionStart",
      session_id: event.properties.sessionID,
      source: "compact",
      cwd: directory,
    });
  },

  "tool.execute.before": async (input, output) => {
    if (
      process.env.TRELLIS_HOOKS === "0" ||
      process.env.TRELLIS_DISABLE_HOOKS === "1"
    ) {
      return;
    }

    const payload = buildToolPayload(directory, input, output);
    if (!payload) return;

    const result = runSpecHook(directory, payload);
    const context = result?.additionalContext;
    if (result?.permissionDecision !== "deny" || typeof context !== "string") {
      return;
    }

    throw new Error(
      "Trellis loaded governing specs before this tool call. " +
        "Follow them, then retry the same tool call.\n\n" +
        context,
    );
  },
});
