"use client";

import { startTransition, useEffect, useState } from "react";
import {
	Activity,
	ArrowUpRight,
	BrainCircuit,
	Flame,
	Radio,
	ScanSearch,
	Sparkles,
} from "lucide-react";

type Insight = {
	id: number;
	platform: string;
	channel: string;
	content_kind: string;
	cluster_key: string;
	cluster_label: string;
	source_post_id: string;
	title: string;
	source_url: string;
	author: string;
	raw_content: string;
	core_pain_point: string;
	current_workaround: string;
	commercial_potential: number;
	saas_feasibility: string;
	is_explicit_content: boolean;
	matched_keywords: string[];
	analysis_model: string;
	embedding_model: string;
	published_at: string;
	created_at: string;
	similarity?: number;
};

type InsightStats = {
	total_insights: number;
	live_last_24h: number;
	average_potential: number;
	top_platform: string;
};

type TrendCluster = {
	cluster_key: string;
	cluster_label: string;
	insight_count: number;
	average_score: number;
	top_score: number;
	latest_seen_at: string;
	platforms: string[];
	sample_titles: string[];
	sample_pain_points: string[];
};

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export default function DashboardPage() {
	const [insights, setInsights] = useState<Insight[]>([]);
	const [stats, setStats] = useState<InsightStats | null>(null);
	const [trends, setTrends] = useState<TrendCluster[]>([]);
	const [semanticQuery, setSemanticQuery] = useState("");
	const [searchResults, setSearchResults] = useState<Insight[]>([]);
	const [searchState, setSearchState] = useState<"idle" | "loading" | "ready" | "error">("idle");
	const [searchError, setSearchError] = useState<string | null>(null);
	const [connectionState, setConnectionState] = useState<"connecting" | "live" | "retrying">("connecting");
	const [lastEventAt, setLastEventAt] = useState<string | null>(null);

	useEffect(() => {
		let isMounted = true;

		const loadSnapshot = async () => {
			try {
				const [insightsResponse, statsResponse, trendsResponse] = await Promise.all([
					fetch(`${API_BASE_URL}/api/insights?limit=40`, { cache: "no-store" }),
					fetch(`${API_BASE_URL}/api/stats`, { cache: "no-store" }),
					fetch(`${API_BASE_URL}/api/trends?limit=8&window_hours=168`, { cache: "no-store" }),
				]);

				if (!insightsResponse.ok || !statsResponse.ok || !trendsResponse.ok) {
					throw new Error("failed to fetch dashboard snapshot");
				}

				const nextInsights = (await insightsResponse.json()) as Insight[];
				const nextStats = (await statsResponse.json()) as InsightStats;
				const nextTrends = (await trendsResponse.json()) as TrendCluster[];

				if (!isMounted) {
					return;
				}

				setInsights(nextInsights);
				setStats(nextStats);
				setTrends(nextTrends);
			} catch (error) {
				console.error("Failed to load snapshot", error);
			}
		};

		loadSnapshot();
		const interval = window.setInterval(loadSnapshot, 60000);

		return () => {
			isMounted = false;
			window.clearInterval(interval);
		};
	}, []);

	useEffect(() => {
		const stream = new EventSource(`${API_BASE_URL}/api/stream`);

		stream.onopen = () => {
			setConnectionState("live");
		};

		stream.addEventListener("insight", (event) => {
			try {
				const nextInsight = JSON.parse(event.data) as Insight;
				startTransition(() => {
					setInsights((current) => {
						const withoutDuplicate = current.filter((item) => item.id !== nextInsight.id);
						return [nextInsight, ...withoutDuplicate].slice(0, 60);
					});
				});
				setLastEventAt(new Date().toISOString());
				setStats((current) => {
					if (!current) {
						return current;
					}
					const total = current.total_insights + 1;
					const average =
						(current.average_potential * current.total_insights + nextInsight.commercial_potential) / total;
					return {
						...current,
						total_insights: total,
						live_last_24h: current.live_last_24h + 1,
						average_potential: Number(average.toFixed(1)),
					};
				});
			} catch (error) {
				console.error("Failed to parse SSE insight", error);
			}
		});

		stream.addEventListener("ping", () => {
			setConnectionState("live");
		});

		stream.onerror = () => {
			setConnectionState("retrying");
		};

		return () => {
			stream.close();
		};
	}, []);

	const featured = insights.slice(0, 3);

	const handleSemanticSearch = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();

		const query = semanticQuery.trim();
		if (!query) {
			setSearchState("idle");
			setSearchResults([]);
			setSearchError(null);
			return;
		}

		setSearchState("loading");
		setSearchError(null);

		try {
			const response = await fetch(`${API_BASE_URL}/api/search?q=${encodeURIComponent(query)}&limit=6`, {
				cache: "no-store",
			});
			if (!response.ok) {
				throw new Error("semantic search failed");
			}
			const results = (await response.json()) as Insight[];
			startTransition(() => setSearchResults(results));
			setSearchState("ready");
		} catch (error) {
			console.error("Semantic search failed", error);
			setSearchState("error");
			setSearchError("Search is unavailable right now. Check your embedder config and API keys.");
		}
	};

	return (
		<main className="min-h-screen">
			<section className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-10 px-5 py-8 sm:px-8 lg:px-10">
				<div className="grid gap-6 lg:grid-cols-[1.3fr_0.7fr]">
					<div className="panel panel-hero overflow-hidden">
						<div className="hero-grid" />
						<div className="relative z-10 flex flex-col gap-6">
							<div className="flex flex-wrap items-center gap-3">
								<span className="eyebrow">
									<ScanSearch size={14} />
									Pain-point radar
								</span>
								<span className={`status-pill status-${connectionState}`}>
									<Radio size={14} />
									{connectionLabel(connectionState)}
								</span>
							</div>

							<div className="space-y-4">
								<h1 className="max-w-4xl text-4xl font-semibold tracking-tight text-white sm:text-5xl lg:text-6xl">
									The Idea Engine turns messy complaints into live SaaS signal.
								</h1>
								<p className="max-w-2xl text-base leading-7 text-slate-300 sm:text-lg">
									用 Go + Kafka 持續抓取 Dcard、Reddit、App Store 與 transcript feeds，把「抱怨」
									轉成可排序、可追蹤、可商業化的 product intelligence。
								</p>
							</div>

							<div className="grid gap-4 sm:grid-cols-3">
								<MetricCard
									label="Total insights"
									value={stats ? `${stats.total_insights}` : "--"}
									helper="Structured pain points saved in Postgres"
									icon={<Activity size={18} />}
								/>
								<MetricCard
									label="Last 24h"
									value={stats ? `${stats.live_last_24h}` : "--"}
									helper="Fresh opportunities pushed through the stream"
									icon={<Sparkles size={18} />}
								/>
								<MetricCard
									label="Avg. potential"
									value={stats ? `${stats.average_potential.toFixed(1)}/10` : "--"}
									helper={stats?.top_platform ? `Top source: ${stats.top_platform}` : "Waiting for source mix"}
									icon={<Flame size={18} />}
								/>
							</div>
						</div>
					</div>

					<div className="panel flex flex-col justify-between gap-6">
						<div className="space-y-3">
							<p className="eyebrow">
								<BrainCircuit size={14} />
								Operating model
							</p>
							<h2 className="text-2xl font-semibold text-white">From ingestion to monetizable pattern</h2>
							<p className="text-sm leading-6 text-slate-300">
								Focus on non-technical users doing clumsy manual work. Those workarounds are usually where
								the blue-ocean SaaS opportunities hide.
							</p>
						</div>

						<div className="space-y-4 text-sm text-slate-300">
							<PipelineStep step="01" title="Ingestion">
								Dcard, Reddit, App Store RSS and transcript feeds normalize into one stream.
							</PipelineStep>
							<PipelineStep step="02" title="Intelligence">
								Gemini, Groq or mock analysis extracts pain point, workaround, score and feasibility.
							</PipelineStep>
							<PipelineStep step="03" title="Presentation">
								SSE pushes fresh insights into a live dashboard so you can watch new demand show up.
							</PipelineStep>
						</div>

						<div className="rounded-2xl border border-white/10 bg-white/5 p-4 font-mono text-xs text-slate-300">
							<p>API</p>
							<p className="mt-2 break-all text-emerald-300">{API_BASE_URL}/api/stream</p>
							{lastEventAt ? (
								<p className="mt-2 text-slate-400">Last event: {formatRelativeTime(lastEventAt)}</p>
							) : (
								<p className="mt-2 text-slate-400">Waiting for the next event...</p>
							)}
						</div>
					</div>
				</div>

					<section className="grid gap-6 xl:grid-cols-[0.72fr_1fr]">
					<div className="flex flex-col gap-6">
						<div className="panel flex flex-col gap-4">
							<div className="flex items-center justify-between">
								<div>
									<p className="eyebrow">
										<Flame size={14} />
										Featured
									</p>
									<h2 className="mt-2 text-2xl font-semibold text-white">Highest-signal ideas right now</h2>
								</div>
							</div>

							<div className="grid gap-4">
								{featured.length === 0 ? (
									<EmptyState />
								) : (
									featured.map((insight) => <FeaturedInsightCard insight={insight} key={insight.id} />)
								)}
							</div>
						</div>

						<div className="panel flex flex-col gap-4">
							<div>
								<p className="eyebrow">
									<Sparkles size={14} />
									Recurring themes
								</p>
								<h2 className="mt-2 text-2xl font-semibold text-white">Pain points that keep showing up</h2>
								<p className="mt-2 text-sm leading-6 text-slate-400">
									這一區把近 7 天內的 insight 聚成重複主題，幫你從單點抱怨往市場需求靠近。
								</p>
							</div>

							<div className="grid gap-3">
								{trends.length === 0 ? (
									<EmptyState />
								) : (
									trends.map((trend) => <TrendCard key={trend.cluster_key} trend={trend} />)
								)}
							</div>
						</div>

						<div className="panel flex flex-col gap-4">
							<div>
								<p className="eyebrow">
									<ScanSearch size={14} />
									Semantic explorer
								</p>
								<h2 className="mt-2 text-2xl font-semibold text-white">Find adjacent pain points by meaning</h2>
								<p className="mt-2 text-sm leading-6 text-slate-400">
									輸入一句問題或 workflow，系統會用 embeddings 找出語意相近的痛點，而不是只做關鍵字比對。
								</p>
							</div>

							<form className="grid gap-3" onSubmit={handleSemanticSearch}>
								<textarea
									className="min-h-28 rounded-3xl border border-white/10 bg-black/20 px-4 py-4 text-sm text-white outline-none placeholder:text-slate-500 focus:border-emerald-300/40"
									onChange={(event) => setSemanticQuery(event.target.value)}
									placeholder="例：餐廳老闆每週都要手動整理外送平台營收，還要再貼到 Excel 做報表"
									value={semanticQuery}
								/>
								<div className="flex items-center justify-between gap-3">
									<p className="font-mono text-xs uppercase tracking-[0.22em] text-slate-500">
										{searchState === "ready" ? `${searchResults.length} matches` : "Ready for vector search"}
									</p>
									<button
										className="rounded-full bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-emerald-300 disabled:cursor-not-allowed disabled:opacity-60"
										disabled={searchState === "loading"}
										type="submit"
									>
										{searchState === "loading" ? "Searching..." : "Search"}
									</button>
								</div>
							</form>

							<div className="grid gap-3">
								{searchError ? <SearchStateCard>{searchError}</SearchStateCard> : null}
								{searchState === "idle" ? (
									<SearchStateCard>
										Start with a workflow description, pain point, or customer quote to surface similar signals.
									</SearchStateCard>
								) : null}
								{searchState === "ready" && searchResults.length === 0 ? (
									<SearchStateCard>No semantic matches yet. Add more source data or try a broader phrasing.</SearchStateCard>
								) : null}
								{searchResults.map((result) => (
									<SearchResultCard key={`search-${result.id}`} result={result} />
								))}
							</div>
						</div>
					</div>

					<div className="panel">
						<div className="mb-5 flex items-center justify-between gap-4">
							<div>
								<p className="eyebrow">
									<Radio size={14} />
									Live pain stream
								</p>
								<h2 className="mt-2 text-2xl font-semibold text-white">Incoming complaints, reviews and hacks</h2>
							</div>
							<div className="rounded-full border border-white/10 bg-white/5 px-3 py-2 font-mono text-xs text-slate-300">
								{insights.length} items buffered
							</div>
						</div>

						<div className="grid gap-4">
							{insights.length === 0 ? (
								<EmptyState />
							) : (
								insights.map((insight) => <InsightCard insight={insight} key={insight.id} />)
							)}
						</div>
					</div>
				</section>
			</section>
		</main>
	);
}

function MetricCard({
	label,
	value,
	helper,
	icon,
}: {
	label: string;
	value: string;
	helper: string;
	icon: React.ReactNode;
}) {
	return (
		<div className="rounded-3xl border border-white/10 bg-black/20 p-4 backdrop-blur">
			<div className="flex items-center justify-between text-slate-300">
				<span className="font-mono text-xs uppercase tracking-[0.3em]">{label}</span>
				<span className="text-emerald-300">{icon}</span>
			</div>
			<p className="mt-4 text-3xl font-semibold text-white">{value}</p>
			<p className="mt-2 text-sm leading-6 text-slate-400">{helper}</p>
		</div>
	);
}

function FeaturedInsightCard({ insight }: { insight: Insight }) {
	return (
		<article className="rounded-3xl border border-emerald-400/20 bg-emerald-400/[0.07] p-5 shadow-[0_0_0_1px_rgba(16,185,129,0.08)]">
			<div className="flex flex-wrap items-center gap-2">
				<SourceBadge value={insight.platform} />
				<SourceBadge value={insight.channel || insight.content_kind} subtle />
				<div className="rounded-full bg-amber-400/15 px-3 py-1 font-mono text-xs text-amber-300">
					Score {insight.commercial_potential}/10
				</div>
			</div>
			<h3 className="mt-4 text-xl font-semibold text-white">{insight.core_pain_point}</h3>
			<p className="mt-3 text-sm leading-7 text-slate-300">{truncate(insight.current_workaround, 180)}</p>
			<p className="mt-4 font-mono text-xs uppercase tracking-[0.25em] text-slate-400">
				{formatRelativeTime(insight.created_at)}
			</p>
		</article>
	);
}

function InsightCard({ insight }: { insight: Insight }) {
	return (
		<article className="rounded-3xl border border-white/10 bg-white/[0.03] p-5 transition duration-300 hover:border-emerald-300/30 hover:bg-white/[0.05]">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div className="flex flex-wrap items-center gap-2">
					<SourceBadge value={insight.platform} />
					<SourceBadge value={insight.channel || insight.content_kind} subtle />
					{insight.matched_keywords?.slice(0, 3).map((keyword) => (
						<span className="rounded-full border border-white/10 px-2.5 py-1 font-mono text-[11px] text-slate-300" key={keyword}>
							{keyword}
						</span>
					))}
				</div>
				<div className="rounded-full bg-amber-400/15 px-3 py-1 font-mono text-xs text-amber-300">
					{insight.commercial_potential}/10
				</div>
			</div>

			<div className="mt-4 grid gap-4 lg:grid-cols-[1.15fr_0.85fr]">
				<div>
					<h3 className="text-lg font-semibold text-white">{insight.core_pain_point}</h3>
					{insight.title ? <p className="mt-2 text-sm text-slate-400">Source: {insight.title}</p> : null}
					<p className="mt-4 text-sm leading-7 text-slate-300">{truncate(insight.raw_content, 260)}</p>
				</div>

				<div className="rounded-2xl border border-white/10 bg-black/20 p-4">
					<p className="font-mono text-xs uppercase tracking-[0.25em] text-slate-400">Current workaround</p>
					<p className="mt-3 text-sm leading-7 text-slate-300">{truncate(insight.current_workaround, 180)}</p>
					<p className="mt-4 font-mono text-xs uppercase tracking-[0.25em] text-slate-400">SaaS feasibility</p>
					<p className="mt-2 text-sm leading-7 text-emerald-200">{insight.saas_feasibility}</p>
				</div>
			</div>

			<div className="mt-5 flex flex-wrap items-center justify-between gap-3">
				<p className="font-mono text-xs uppercase tracking-[0.25em] text-slate-500">
					{formatRelativeTime(insight.created_at)}
				</p>
				<a
					className="inline-flex items-center gap-2 text-sm text-emerald-300 transition hover:text-emerald-200"
					href={insight.source_url}
					rel="noreferrer"
					target="_blank"
				>
					Open source
					<ArrowUpRight size={16} />
				</a>
			</div>
		</article>
	);
}

function TrendCard({ trend }: { trend: TrendCluster }) {
	const sample = trend.sample_pain_points[0] || trend.sample_titles[0] || trend.cluster_label;

	return (
		<article className="rounded-3xl border border-white/10 bg-white/[0.03] p-4">
			<div className="flex items-start justify-between gap-3">
				<div>
					<h3 className="text-base font-semibold text-white">{trend.cluster_label}</h3>
					<p className="mt-2 text-sm leading-6 text-slate-400">{truncate(sample, 120)}</p>
				</div>
				<div className="rounded-2xl bg-emerald-400/12 px-3 py-2 text-right">
					<p className="font-mono text-[11px] uppercase tracking-[0.2em] text-emerald-300">Density</p>
					<p className="mt-1 text-lg font-semibold text-white">{trend.insight_count}</p>
				</div>
			</div>

			<div className="mt-4 flex flex-wrap items-center gap-2">
				<span className="rounded-full bg-amber-400/15 px-3 py-1 font-mono text-[11px] text-amber-300">
					Avg {trend.average_score.toFixed(1)}
				</span>
				<span className="rounded-full border border-white/10 px-3 py-1 font-mono text-[11px] text-slate-300">
					Peak {trend.top_score}/10
				</span>
				{trend.platforms.slice(0, 3).map((platform) => (
					<SourceBadge key={platform} subtle value={platform} />
				))}
			</div>

			<p className="mt-4 font-mono text-xs uppercase tracking-[0.25em] text-slate-500">
				Last seen {formatRelativeTime(trend.latest_seen_at)}
			</p>
		</article>
	);
}

function SearchResultCard({ result }: { result: Insight }) {
	return (
		<article className="rounded-3xl border border-white/10 bg-white/[0.03] p-4">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div className="flex flex-wrap items-center gap-2">
					<SourceBadge value={result.platform} />
					{result.cluster_label ? <SourceBadge subtle value={result.cluster_label} /> : null}
				</div>
				<div className="rounded-full bg-sky-400/15 px-3 py-1 font-mono text-[11px] text-sky-300">
					Similarity {formatSimilarity(result.similarity)}
				</div>
			</div>

			<h3 className="mt-4 text-base font-semibold text-white">{result.core_pain_point}</h3>
			<p className="mt-2 text-sm leading-6 text-slate-400">{truncate(result.current_workaround, 140)}</p>
			<div className="mt-4 flex items-center justify-between gap-3">
				<p className="font-mono text-xs uppercase tracking-[0.2em] text-slate-500">
					{result.embedding_model || "vector search"}
				</p>
				<a
					className="inline-flex items-center gap-2 text-sm text-emerald-300 transition hover:text-emerald-200"
					href={result.source_url}
					rel="noreferrer"
					target="_blank"
				>
					Open source
					<ArrowUpRight size={16} />
				</a>
			</div>
		</article>
	);
}

function SearchStateCard({ children }: { children: React.ReactNode }) {
	return <div className="rounded-3xl border border-dashed border-white/10 bg-black/20 p-5 text-sm leading-6 text-slate-400">{children}</div>;
}

function PipelineStep({
	step,
	title,
	children,
}: {
	step: string;
	title: string;
	children: React.ReactNode;
}) {
	return (
		<div className="flex gap-4">
			<div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-white/10 bg-black/25 font-mono text-xs text-emerald-300">
				{step}
			</div>
			<div>
				<p className="text-sm font-semibold text-white">{title}</p>
				<p className="mt-1 leading-6 text-slate-400">{children}</p>
			</div>
		</div>
	);
}

function SourceBadge({ value, subtle = false }: { value: string; subtle?: boolean }) {
	return (
		<span
			className={`rounded-full px-3 py-1 font-mono text-[11px] uppercase tracking-[0.22em] ${
				subtle ? "border border-white/10 text-slate-300" : "bg-emerald-400/15 text-emerald-300"
			}`}
		>
			{value || "unknown"}
		</span>
	);
}

function EmptyState() {
	return (
		<div className="rounded-3xl border border-dashed border-white/10 bg-black/20 p-8 text-center text-slate-400">
			目前還沒有新的 pain point 進來。先啟動 backend、Kafka、Redis 和 Postgres，然後讓 ingestion loop 跑一輪。
		</div>
	);
}

function connectionLabel(state: "connecting" | "live" | "retrying") {
	if (state === "live") {
		return "Live stream active";
	}
	if (state === "retrying") {
		return "Reconnecting";
	}
	return "Connecting";
}

function formatRelativeTime(value: string) {
	const date = new Date(value);
	const deltaMs = Date.now() - date.getTime();
	const deltaMinutes = Math.floor(deltaMs / 60000);

	if (Number.isNaN(deltaMinutes)) {
		return "unknown";
	}
	if (deltaMinutes < 1) {
		return "just now";
	}
	if (deltaMinutes < 60) {
		return `${deltaMinutes}m ago`;
	}

	const deltaHours = Math.floor(deltaMinutes / 60);
	if (deltaHours < 24) {
		return `${deltaHours}h ago`;
	}

	const deltaDays = Math.floor(deltaHours / 24);
	return `${deltaDays}d ago`;
}

function truncate(value: string, maxLength: number) {
	if (!value) {
		return "No additional source context available yet.";
	}
	if (value.length <= maxLength) {
		return value;
	}
	return `${value.slice(0, maxLength).trim()}...`;
}

function formatSimilarity(value?: number) {
	if (typeof value !== "number") {
		return "--";
	}
	return `${Math.round(value * 100)}%`;
}
