import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";

import { HeroTerminalDemo } from "../hero-terminal-demo";

afterEach(() => {
  vi.useRealTimers();
});

it("shows image edits and video request examples", () => {
  vi.useFakeTimers();
  render(<HeroTerminalDemo />);

  expect(screen.getByRole("button", { name: "Images" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Video$/ })).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Images" }));
  act(() => vi.advanceTimersByTime(220));
  expect(screen.getByText("/v1/images/edits")).toBeInTheDocument();
  expect(screen.getByText('model="your-image-model"')).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: /Video$/ }));
  act(() => vi.advanceTimersByTime(220));
  expect(screen.getByText("/v1/videos")).toBeInTheDocument();
  expect(document.body.textContent).toContain("video-v1");
});
