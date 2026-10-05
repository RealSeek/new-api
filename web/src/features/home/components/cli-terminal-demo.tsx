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
import { ClaudeCode, Codex, GeminiCLI } from "@lobehub/icons";
import { Pause, Play, RotateCcw, Terminal } from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { CopyButton } from "@/components/copy-button";
import { Button } from "@/components/ui/button";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import "@/styles/cli-terminal.css";

import { CLI_DEMO_SCENARIOS, CLI_TOOL_NAMES } from "./cli-demo-scenarios";

const CLIENTS = [
  {
    id: "claude",
    label: "Claude Code",
    icon: ClaudeCode.Color,
    doc: "claude-code",
    prompt: "❯",
    bullet: "●",
  },
  {
    id: "codex",
    label: "Codex CLI",
    icon: Codex,
    doc: "codex",
    prompt: "›",
    bullet: "•",
  },
  {
    id: "gemini",
    label: "Gemini CLI",
    icon: GeminiCLI.Color,
    doc: "gemini-cli",
    prompt: ">",
    bullet: "✦",
  },
  {
    id: "pi",
    label: "pi CLI",
    icon: Terminal,
    doc: "api-overview",
    prompt: "",
    bullet: "",
  },
] as const;

type CliClient = (typeof CLIENTS)[number];

export function CliTerminalDemo() {
  const { t } = useTranslation();
  const [clientId, setClientId] = useState<string>("claude");

  return (
    <Tabs
      value={clientId}
      onValueChange={(value) => setClientId(String(value))}
      className="min-w-0 gap-3"
    >
      <TabsList
        variant="line"
        aria-label={t("CLI client")}
        className="grid w-full grid-cols-4 gap-0 px-0 group-data-horizontal/tabs:h-12"
      >
        {CLIENTS.map((client) => (
          <TabsTrigger
            key={client.id}
            value={client.id}
            className="h-full min-w-0 gap-1 px-1 text-[11px] group-data-horizontal/tabs:after:bottom-0 sm:gap-1.5 sm:text-xs"
          >
            <client.icon size={16} aria-hidden className="shrink-0" />
            <span className="whitespace-normal">{client.label}</span>
          </TabsTrigger>
        ))}
      </TabsList>
      {CLIENTS.map((client) => (
        <TabsContent key={client.id} value={client.id}>
          {clientId === client.id && (
            <CliSession key={client.id} client={client} />
          )}
        </TabsContent>
      ))}
    </Tabs>
  );
}

function CliWelcome(props: { client: CliClient }) {
  const { t } = useTranslation();
  switch (props.client.id) {
    case "claude":
      return (
        <div className="cli-welcome cli-claude-welcome">
          <pre className="cli-claude-logo" aria-hidden>
            {" ▐▛███▜▌\n▝▜█████▛▘\n  ▘▘ ▝▝"}
          </pre>
          <div>
            <strong>Claude Code</strong>
            <div className="cli-muted">Claude · {t("Local demo")}</div>
            <div className="cli-muted">~/onlycode</div>
          </div>
        </div>
      );
    case "codex":
      return (
        <div className="cli-welcome cli-codex-welcome">
          <div>
            <span className="cli-muted" aria-hidden>
              {">_"}
            </span>{" "}
            <strong>OpenAI Codex</strong>
          </div>
          <div className="mt-3 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4">
            <span className="cli-muted">{t("Model")}:</span>
            <span>{t("Local demo")}</span>
            <span className="cli-muted">cwd:</span>
            <span>~/onlycode</span>
          </div>
        </div>
      );
    case "gemini":
      return (
        <div className="cli-welcome">
          {/* Official wordmark: Google LLC, Apache-2.0, gemini-cli/AsciiArt.ts (shortAsciiLogo). */}
          <pre className="cli-gemini-logo" aria-label="Gemini">
            {
              "   █████████  ██████████ ██████   ██████ █████ ██████   █████ █████\n  ███░░░░░███░░███░░░░░█░░██████ ██████ ░░███ ░░██████ ░░███ ░░███\n ███     ░░░  ░███  █ ░  ░███░█████░███  ░███  ░███░███ ░███  ░███\n░███          ░██████    ░███░░███ ░███  ░███  ░███░░███░███  ░███\n░███    █████ ░███░░█    ░███ ░░░  ░███  ░███  ░███ ░░██████  ░███\n░░███  ░░███  ░███ ░   █ ░███      ░███  ░███  ░███  ░░█████  ░███\n ░░█████████  ██████████ █████     █████ █████ █████  ░░█████ █████\n  ░░░░░░░░░  ░░░░░░░░░░ ░░░░░     ░░░░░ ░░░░░ ░░░░░    ░░░░░ ░░░░░"
            }
          </pre>
        </div>
      );
    case "pi":
      return (
        <div className="cli-welcome cli-pi-welcome">
          <div className="cli-pi-logo" aria-hidden>
            <div>
              <span className="cli-pi-coral-blue">▀</span>
              <span className="cli-pi-coral">▀█</span>{" "}
            </div>
            <div>
              <span className="cli-pi-blue">█▀</span>{" "}
              <span className="cli-pi-yellow">█</span>
            </div>
          </div>
          <strong>pi</strong>
          <span className="cli-muted">{t("Local demo")}</span>
        </div>
      );
  }
}

function CliSession(props: { client: CliClient }) {
  const { t } = useTranslation();
  const reduceMotion =
    useReducedMotion() && !import.meta.env.MODE.includes("test");
  const [playback, setPlayback] = useState({ index: 0, elapsed: 0 });
  const [paused, setPaused] = useState(false);
  const transcriptRef = useRef<HTMLDivElement>(null);
  const followOutputRef = useRef(true);
  const scene = CLI_DEMO_SCENARIOS[playback.index];
  const question = t(scene.questionKey);
  const answer = t(scene.answerKey);
  const promptEnd = [...question].length * 45 + 600;
  const thinkingEnd = promptEnd + 2400;
  const toolsEnd = thinkingEnd + scene.tools.length * 1600;
  const answerEnd = toolsEnd + [...answer].length * 28;
  const cycleEnd = answerEnd + 5000;
  const elapsed = reduceMotion ? answerEnd : playback.elapsed;
  const isPrompting = elapsed < promptEnd;
  const isThinking = !isPrompting && elapsed < thinkingEnd;
  const isRunningTools = elapsed >= thinkingEnd && elapsed < toolsEnd;
  const isAnswering = elapsed >= toolsEnd && elapsed < answerEnd;
  const visibleQuestion = [...question]
    .slice(0, Math.floor(elapsed / 45))
    .join("");
  const visibleAnswer = [...answer]
    .slice(0, Math.max(0, Math.floor((elapsed - toolsEnd) / 28)))
    .join("");
  const shownTools = Math.min(
    scene.tools.length,
    Math.max(0, Math.floor((elapsed - thinkingEnd) / 1600) + 1),
  );
  const stageLabel = isThinking ? t("Thinking...") : t("Running");
  const playing = !paused && !reduceMotion;

  useEffect(() => {
    followOutputRef.current = true;
  }, [playback.index]);

  useEffect(() => {
    if (!playing) return;
    const timer = setInterval(() => {
      if (document.hidden) return;
      setPlayback((current) => {
        if (current.elapsed + 40 < cycleEnd) {
          return { ...current, elapsed: current.elapsed + 40 };
        }
        // A random nonzero offset prevents consecutive repeats.
        const offset =
          1 + Math.floor(Math.random() * (CLI_DEMO_SCENARIOS.length - 1));
        return {
          index: (current.index + offset) % CLI_DEMO_SCENARIOS.length,
          elapsed: 0,
        };
      });
    }, 40);
    return () => clearInterval(timer);
  }, [playing, playback.index, cycleEnd]);

  useEffect(() => {
    const transcript = transcriptRef.current;
    if (!transcript || !followOutputRef.current) return;
    transcript.scrollTop = isPrompting ? 0 : transcript.scrollHeight;
  }, [elapsed, isPrompting]);

  function replay() {
    setPlayback((current) => ({
      index: reduceMotion
        ? (current.index + 1) % CLI_DEMO_SCENARIOS.length
        : current.index,
      elapsed: 0,
    }));
    setPaused(false);
    followOutputRef.current = true;
  }

  let status = t("Done");
  if (isPrompting) status = t("Question");
  else if (isThinking) status = t("Thinking...");
  else if (isRunningTools) status = t("Running");
  else if (isAnswering) status = t("Writing…");
  if (paused) status = t("Playback paused");

  return (
    <div
      className="cli-screen"
      data-cli={props.client.id}
      data-playing={playing}
    >
      <div className="cli-tools">
        <span>{t("Local demo")}</span>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className="cli-control size-6"
                aria-label={
                  playing ? t("Pause playback") : t("Resume playback")
                }
                disabled={!!reduceMotion}
                onClick={() => setPaused((current) => !current)}
              >
                {playing ? (
                  <Pause className="size-3.5" />
                ) : (
                  <Play className="size-3.5" />
                )}
              </Button>
            }
          />
          <TooltipContent>
            {playing ? t("Pause playback") : t("Resume playback")}
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className="cli-control size-6"
                aria-label={t("Replay demo")}
                onClick={replay}
              >
                <RotateCcw className="size-3.5" />
              </Button>
            }
          />
          <TooltipContent>{t("Replay demo")}</TooltipContent>
        </Tooltip>
      </div>
      <div
        ref={transcriptRef}
        className="cli-transcript"
        aria-label={t("Terminal output")}
        onScroll={(event) => {
          const element = event.currentTarget;
          followOutputRef.current =
            element.scrollHeight - element.scrollTop - element.clientHeight <
            24;
        }}
      >
        <CliWelcome client={props.client} />
        {!isPrompting && (
          <>
            <div className="cli-question">
              <span className="cli-prompt" aria-hidden>
                {props.client.prompt}
              </span>
              <span className="min-w-0 break-words">{question}</span>
            </div>
            {(isThinking || isRunningTools) && (
              <div className="cli-thinking" aria-label={t("Thinking...")}>
                <span className="cli-spinner" aria-hidden>
                  {props.client.id === "claude" ? "✻" : "⠋"}
                </span>
                <div>
                  <span className="cli-thinking-label">{stageLabel}</span>
                  <div className="cli-thinking-summary">
                    {t(scene.thinkingKey)}
                  </div>
                </div>
              </div>
            )}
            {scene.tools.slice(0, shownTools).map((tool, index) => {
              const complete = elapsed >= thinkingEnd + (index + 1) * 1600;
              return (
                <div
                  key={tool.kind + tool.target}
                  className="cli-tool-call"
                  data-complete={complete}
                  aria-label={CLI_TOOL_NAMES[props.client.id][tool.kind]}
                >
                  <div className="cli-tool-title">
                    <span aria-hidden>{complete ? "✓" : "⠋"}</span>
                    <strong>
                      {CLI_TOOL_NAMES[props.client.id][tool.kind]}
                    </strong>
                    <span>{tool.target}</span>
                  </div>
                  {complete && (
                    <pre className="cli-tool-output">{tool.output}</pre>
                  )}
                </div>
              );
            })}
            {elapsed >= toolsEnd && (
              <div className="cli-response">
                <span className="cli-bullet" aria-hidden>
                  {props.client.bullet}
                </span>
                <pre
                  aria-label={t("Response")}
                  aria-busy={isAnswering}
                  className="m-0 min-w-0 flex-1 break-words whitespace-pre-wrap"
                >
                  {visibleAnswer}
                  {isAnswering && (
                    <span
                      aria-hidden
                      className="cli-cursor ml-0.5 inline-block h-3 w-1.5 motion-safe:animate-pulse"
                    />
                  )}
                </pre>
                {!isAnswering && (
                  <CopyButton
                    value={answer}
                    className="cli-control size-6 shrink-0"
                    tooltip={t("Copy to clipboard")}
                  />
                )}
              </div>
            )}
          </>
        )}
      </div>
      <div className="cli-form">
        <InputGroup className="cli-input">
          <InputGroupAddon>
            <span aria-hidden className="cli-prompt">
              {props.client.prompt}
            </span>
          </InputGroupAddon>
          <InputGroupInput
            readOnly
            tabIndex={-1}
            value={isPrompting ? visibleQuestion : ""}
            aria-label={t("Question")}
            placeholder={t("Ask a question…")}
            className="min-w-0 text-xs"
          />
        </InputGroup>
      </div>
      <div className="cli-footer">
        <span>~/onlycode</span>
        <span>
          {props.client.label} · {t("Local demo")}
        </span>
      </div>
      <span className="sr-only" role="status">
        {status}
      </span>
    </div>
  );
}
