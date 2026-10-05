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
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { CliTerminalDemo } from "../cli-terminal-demo";

beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(Math, "random").mockReturnValue(0);
});

afterEach(() => {
  vi.clearAllTimers();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

it("plays a question, thinking, file operations and a final summary in order", () => {
  render(<CliTerminalDemo />);
  const input = screen.getByRole("textbox", { name: "Question" });
  expect(input).toHaveAttribute("readonly");
  act(() => vi.advanceTimersByTime(800));
  expect(input).toHaveValue("Create a Python H");
  act(() => vi.advanceTimersByTime(2000));
  expect(screen.getByRole("status")).toHaveTextContent("Thinking...");
  expect(screen.getByLabelText("Thinking...")).toHaveTextContent(
    "check its output",
  );
  act(() => vi.advanceTimersByTime(2400));
  expect(screen.getByLabelText("Write")).toHaveTextContent("hello.py");
  act(() => vi.advanceTimersByTime(1600));
  expect(screen.getByLabelText("Write")).toHaveTextContent(
    'print("Hello, world!")',
  );
  expect(screen.getByLabelText("Bash")).toHaveTextContent("python hello.py");
  act(() => vi.advanceTimersByTime(1200));
  expect(screen.getByRole("status")).toHaveTextContent("Running");
  expect(screen.queryByLabelText("Response")).not.toBeInTheDocument();
  act(() => vi.advanceTimersByTime(2400));
  expect(screen.getByRole("status")).toHaveTextContent("Done");
  expect(screen.getByLabelText("Response")).toHaveAttribute(
    "aria-busy",
    "false",
  );
  expect(screen.getByLabelText("Response")).toHaveTextContent(
    "Created hello.py and ran it successfully.",
  );
  expect(screen.getByLabelText("Response")).toHaveAttribute(
    "aria-busy",
    "false",
  );
});

it("pauses at the current frame, resumes and replays from the beginning", () => {
  render(<CliTerminalDemo />);
  act(() => vi.advanceTimersByTime(800));
  fireEvent.click(screen.getByRole("button", { name: "Pause playback" }));
  const input = screen.getByRole("textbox", { name: "Question" });
  const partial = (input as HTMLInputElement).value;
  act(() => vi.advanceTimersByTime(4000));
  expect(input).toHaveValue(partial);
  expect(screen.getByRole("status")).toHaveTextContent("Playback paused");
  fireEvent.click(screen.getByRole("button", { name: "Resume playback" }));
  act(() => vi.advanceTimersByTime(2000));
  expect(screen.getByRole("status")).toHaveTextContent("Thinking...");
  fireEvent.click(screen.getByRole("button", { name: "Replay demo" }));
  expect(input).toHaveValue("");
  expect(screen.queryByLabelText("Write")).not.toBeInTheDocument();
});

it("chooses a different scenario after the completed response has been held", () => {
  render(<CliTerminalDemo />);
  act(() => vi.advanceTimersByTime(16000));
  expect(screen.getByRole("textbox", { name: "Question" })).toHaveValue(
    "Fix the empty-l",
  );
  act(() => vi.advanceTimersByTime(2400));
  expect(screen.getByLabelText("Thinking...")).toHaveTextContent(
    "empty-list case",
  );
});

it.each([
  ["Codex CLI", "Added", "Ran"],
  ["Gemini CLI", "WriteFile", "Shell"],
  ["pi CLI", "write", "bash"],
])(
  "uses %s tool labels and starts a fresh playback after switching",
  (client, write, shell) => {
    render(<CliTerminalDemo />);
    act(() => vi.advanceTimersByTime(800));
    fireEvent.click(screen.getByRole("button", { name: "Pause playback" }));
    fireEvent.click(screen.getByRole("tab", { name: client }));
    expect(screen.getByRole("tab", { name: client })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByRole("textbox", { name: "Question" })).toHaveValue("");
    expect(
      screen.getByRole("button", { name: "Pause playback" }),
    ).toBeEnabled();
    act(() => vi.advanceTimersByTime(8000));
    expect(screen.getByLabelText(write)).toHaveTextContent("hello.py");
    expect(screen.getByLabelText(shell)).toHaveTextContent("python hello.py");
  },
);
