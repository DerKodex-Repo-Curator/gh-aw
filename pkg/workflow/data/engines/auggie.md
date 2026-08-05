---
engine:
  id: auggie
  display-name: Auggie
  description: Augment's agentic coding CLI (experimental)
  experimental: true
  provider:
    name: augment
  auth:
    - role: session
      secret: AUGMENT_SESSION_AUTH
  behaviors:
    supported-env-var-keys:
      - AUGMENT_SESSION_AUTH
    manifest:
      files:
        - AGENTS.md
      path-prefixes:
        - .augment/
    network:
      defaults:
        - host.docker.internal
        - registry.npmjs.org
        - api.augmentcode.com
        - auth.augmentcode.com
        - "*.api.augmentcode.com"
    installation:
      package-manager: npm
      package-name: "@augmentcode/auggie"
      version: "0.34.0"
      step-name: Install Auggie CLI
      binary-name: auggie
      include-node-setup: true
      cooldown: true
      verify-command: auggie --version
      verify-step-name: Verify Auggie CLI installation
      docs-url: https://docs.augmentcode.com/cli/setup-auggie/authentication
    execution:
      command-name: auggie
      args:
        - --print
        - --quiet
        - --output-format
        - json
        - --dont-save-session
        - --no-update-terminal-title
        - --permission
        - remove-files:allow
        - --permission
        - save-file:allow
        - --permission
        - apply_patch:allow
        - --permission
        - str-replace-editor:allow
        - --permission
        - view:allow
        - --permission
        - launch-process:allow
        - --permission
        - kill-process:allow
        - --permission
        - read-process:allow
        - --permission
        - write-process:allow
        - --permission
        - list-processes:allow
        - --permission
        - view-session:allow
        - --permission
        - codebase-retrieval-raw:allow
        - --permission
        - view_tasklist:allow
        - --permission
        - reorganize_tasklist:allow
        - --permission
        - update_tasks:allow
        - --permission
        - add_tasks:allow
        - --permission
        - grep-search:allow
        - --permission
        - view-range-untruncated:allow
        - --permission
        - search-untruncated:allow
        - --permission
        - web-fetch:deny
        - --remove-tool
        - web-fetch
      step-name: Execute Auggie CLI
      mcp-config-env-var: AUGGIE_MCP_CONFIG
      write-timestamp: true
      env:
        AUGMENT_DISABLE_AUTO_UPDATE: "1"
    mcp:
      config-path: /tmp/gh-aw/auggie-mcp.json
    harness-script: |
      const { chmodSync, existsSync, readFileSync } = require("fs");
      const { spawnSync } = require("child_process");

      const [command, ...commandArgs] = process.argv.slice(2);
      const promptFile = process.env.GH_AW_PROMPT;
      const mcpConfig = process.env.AUGGIE_MCP_CONFIG;

      const fail = (message) => {
        process.stderr.write(`[auggie-harness] ${message}\n`);
        process.exitCode = 1;
      };

      if (!command) {
        fail("Auggie command is not configured");
      } else if (!promptFile || !existsSync(promptFile)) {
        fail("GH_AW_PROMPT does not reference a readable prompt file");
      } else {
        const args = [...commandArgs, "--instruction-file", promptFile];

        if (mcpConfig) {
          if (!existsSync(mcpConfig)) {
            fail(`AUGGIE_MCP_CONFIG does not exist: ${mcpConfig}`);
          } else {
            try {
              const config = JSON.parse(readFileSync(mcpConfig, "utf8"));
              if (!config.mcpServers || typeof config.mcpServers !== "object" || Array.isArray(config.mcpServers)) {
                throw new Error("top-level mcpServers object is required");
              }
              for (const [name, server] of Object.entries(config.mcpServers)) {
                if (!server || typeof server !== "object" || (typeof server.command !== "string" && typeof server.url !== "string")) {
                  throw new Error(`MCP server ${name} must define command or url`);
                }
              }
              chmodSync(mcpConfig, 0o600);
              args.push("--mcp-config", mcpConfig);
            } catch (error) {
              fail(`invalid MCP configuration: ${error instanceof Error ? error.message : String(error)}`);
            }
          }
        }

        if (process.exitCode !== 1) {
          const result = spawnSync(command, args, {
            cwd: process.env.GH_AW_ENGINE_CWD || process.env.GITHUB_WORKSPACE,
            env: process.env,
            encoding: "utf8",
            maxBuffer: 64 * 1024 * 1024,
          });

          if (result.stderr) process.stderr.write(result.stderr);
          if (result.signal) {
            process.kill(process.pid, result.signal);
          } else if (result.error) {
            fail(`failed to execute Auggie: ${result.error.message}`);
          } else if (result.status !== 0) {
            if (result.stdout) process.stdout.write(result.stdout);
            process.exitCode = result.status || 1;
          } else {
            try {
              JSON.parse(result.stdout);
              process.stdout.write(result.stdout);
            } catch {
              fail("Auggie exited successfully but did not return valid JSON");
            }
          }
        }
      }
---

<!-- # Auggie

Built-in experimental engine configuration for Augment's Auggie CLI. -->
