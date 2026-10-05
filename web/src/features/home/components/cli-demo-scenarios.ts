/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type DemoTool = {
  kind: "read" | "write" | "edit" | "shell";
  target: string;
  output: string;
};

// These transcripts are fixtures only. No files, commands or model calls are executed.
export const CLI_DEMO_SCENARIOS = [
  {
    questionKey: "Create a Python Hello World script and run it.",
    thinkingKey: "I will create a small script, then check its output.",
    tools: [
      {
        kind: "write",
        target: "hello.py",
        output: '1  print("Hello, world!")',
      },
      { kind: "shell", target: "python hello.py", output: "Hello, world!" },
    ],
    answerKey:
      "Created hello.py and ran it successfully. The script prints Hello, world!",
  },
  {
    questionKey: "Fix the empty-list bug and run the regression test.",
    thinkingKey: "I will inspect the function and test the empty-list case.",
    tools: [
      {
        kind: "read",
        target: "src/average.ts",
        output: "return sum / values.length",
      },
      {
        kind: "edit",
        target: "src/average.ts",
        output: "+ if (values.length === 0) return 0",
      },
      {
        kind: "shell",
        target: "bun test average.test.ts",
        output: "✓ empty list returns 0\n✓ average of [2, 4] is 3\n2 passed",
      },
    ],
    answerKey: "Fixed the empty-list case. Both regression tests pass.",
  },
  {
    questionKey: "Explain how this project routes an API request.",
    thinkingKey: "I will trace the route, controller and upstream adapter.",
    tools: [
      {
        kind: "read",
        target: "router/relay-router.go",
        output: "POST /v1/chat/completions → Relay",
      },
      {
        kind: "read",
        target: "relay/relay.go",
        output: "request → channel → adapter → upstream",
      },
    ],
    answerKey:
      "The router accepts the request, selects a channel and forwards it through the provider adapter. The response returns through the same gateway.",
  },
] satisfies {
  questionKey: string;
  thinkingKey: string;
  tools: DemoTool[];
  answerKey: string;
}[];

export const CLI_TOOL_NAMES = {
  claude: { read: "Read", write: "Write", edit: "Update", shell: "Bash" },
  codex: { read: "Explored", write: "Added", edit: "Updated", shell: "Ran" },
  gemini: {
    read: "ReadFile",
    write: "WriteFile",
    edit: "Edit",
    shell: "Shell",
  },
  pi: { read: "read", write: "write", edit: "edit", shell: "bash" },
} as const;
