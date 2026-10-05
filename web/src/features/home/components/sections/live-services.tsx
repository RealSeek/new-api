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
import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  Image,
  KeyRound,
  MessagesSquare,
  Video,
  WandSparkles,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { AnimateInView } from "@/components/animate-in-view";
import { ErrorState } from "@/components/error-state";
import { Button } from "@/components/ui/button";
import { useGroupStatusBoard } from "@/features/group-status/api";
import { GroupStatusBoard } from "@/features/group-status/components/group-status-board";
import { formatGroupRate } from "@/features/group-status/lib/format";
import {
  formatLatency,
  formatThroughput,
} from "@/features/performance-metrics/lib/format";
import { toIntlLocale } from "@/i18n/languages";
import { formatCompactNumber } from "@/lib/format";
import { getServerErrorMessage } from "@/lib/server-error-message";

import { HeroTerminalDemo } from "../hero-terminal-demo";
import { LiveTopology } from "../live-topology";

export function LiveServices() {
  const { t, i18n } = useTranslation();
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language);
  const board = useGroupStatusBoard();
  const metrics = [
    {
      label: t("Availability"),
      value: formatGroupRate(board.data?.summary?.success_rate, locale),
    },
    {
      label: t("Latency short"),
      value: formatLatency(board.data?.summary?.avg_latency_ms ?? 0),
    },
    {
      label: t("Throughput short"),
      value: formatThroughput(board.data?.summary?.avg_tps ?? 0),
    },
    {
      label: t("24h requests"),
      value: board.data
        ? formatCompactNumber(board.data.request_count, locale)
        : "-",
    },
  ];

  return (
    <>
      <section className="px-4 py-12 sm:px-6 lg:py-16">
        <AnimateInView animation="fade-right" className="mx-auto max-w-6xl">
          <LiveTopology />
        </AnimateInView>
      </section>

      {board.canView && (
        <section
          id="group-status"
          className="scroll-mt-20 px-4 py-12 sm:px-6 lg:py-16"
        >
          <AnimateInView animation="fade-left" className="mx-auto max-w-6xl">
            <dl className="mb-10 grid grid-cols-2 gap-6 border-y py-6 sm:grid-cols-4">
              {metrics.map((metric) => (
                <div key={metric.label}>
                  <dt className="text-muted-foreground text-xs">
                    {metric.label}
                  </dt>
                  <dd className="mt-2 font-mono text-2xl tabular-nums">
                    {metric.value}
                  </dd>
                </div>
              ))}
            </dl>
            <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-semibold">
                {t("Live group status")}
              </h2>
              <Button
                variant="ghost"
                size="sm"
                render={<Link to="/docs/$slug" params={{ slug: "groups" }} />}
              >
                <BookOpen className="size-4" />
                {t("Docs")}
              </Button>
            </div>
            {board.isError ? (
              <ErrorState
                description={getServerErrorMessage(board.error)}
                onRetry={() => {
                  void board.refetch();
                }}
              />
            ) : (
              <GroupStatusBoard
                groups={board.data?.groups ?? []}
                isLoading={board.isLoading}
              />
            )}
          </AnimateInView>
        </section>
      )}
      <section className="border-y px-4 py-12 sm:px-6 lg:py-16">
        <AnimateInView
          animation="fade-right"
          className="mx-auto grid max-w-6xl items-center gap-10 lg:grid-cols-2"
        >
          <div>
            <h2 className="mb-6 text-2xl font-semibold">{t("Quick Start")}</h2>
            <ol className="space-y-5">
              {[
                {
                  slug: "register",
                  title: t("Create an account"),
                  icon: BookOpen,
                },
                {
                  slug: "api-keys",
                  title: t("Create an API key"),
                  icon: KeyRound,
                },
                {
                  slug: "first-request",
                  title: t("Make your first request"),
                  icon: ArrowRight,
                },
              ].map((step, index) => (
                <li key={step.slug}>
                  <Link
                    to="/docs/$slug"
                    params={{ slug: step.slug }}
                    className="group flex items-center gap-4 py-2"
                  >
                    <span className="text-muted-foreground font-mono text-sm">
                      0{index + 1}
                    </span>
                    <step.icon className="size-5" />
                    <span className="flex-1 text-sm font-medium">
                      {step.title}
                    </span>
                    <ArrowRight className="text-muted-foreground size-4 transition-transform group-hover:translate-x-1" />
                  </Link>
                </li>
              ))}
            </ol>
          </div>
          <HeroTerminalDemo />
        </AnimateInView>
      </section>
      <section className="px-4 py-12 sm:px-6 lg:py-16">
        <AnimateInView animation="fade-left" className="mx-auto max-w-6xl">
          <h2 className="mb-6 text-2xl font-semibold">
            {t("Explore the API")}
          </h2>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {[
              {
                slug: "api-overview",
                title: t("Chat & coding"),
                icon: MessagesSquare,
                protocols: "OpenAI / Anthropic / Gemini",
              },
              {
                slug: "images",
                title: t("Image generation"),
                icon: Image,
                protocols: "/v1/images/generations",
              },
              {
                slug: "videos",
                title: t("Video generation"),
                icon: Video,
                protocols: "/v1/videos",
              },
              {
                slug: "workbench",
                title: t("AI workbench"),
                icon: WandSparkles,
                protocols: "OnlyArt",
              },
            ].map((item) => (
              <Link
                key={item.slug}
                to="/docs/$slug"
                params={{ slug: item.slug }}
                className="hover:text-primary border-t py-5 transition-colors"
              >
                <item.icon className="mb-4 size-6" />
                <h3 className="text-sm font-semibold">{item.title}</h3>
                <p className="text-muted-foreground mt-2 font-mono text-xs break-all">
                  {item.protocols}
                </p>
                <ArrowRight className="mt-4 size-4" />
              </Link>
            ))}
          </div>
        </AnimateInView>
      </section>
    </>
  );
}
