"use client";

import { useEffect, useState } from "react";
import { Activity, Flame, ExternalLink, Code2 } from "lucide-react";

// Mock type based on our Go backend insight struct
type Insight = {
  id: number;
  platform: string;
  core_pain_point: string;
  current_workaround: string;
  commercial_potential: number;
  saas_feasibility: string;
  source_url: string;
};

export default function Dashboard() {
  const [insights, setInsights] = useState<Insight[]>([]);

  useEffect(() => {
    // TODO: Connect this to a real Go SSE (Server-Sent Events) endpoint or WebSocket
    // fetch('/api/insights').then(res => res.json()).then(setInsights)

    // Mock data for initial UI layout
    setInsights([
      {
        id: 1,
        platform: "Dcard",
        core_pain_point: "Finding temporary housing with pet-friendly rules is impossible and takes 20+ hours of manual calling.",
        current_workaround: "Using 5 different Facebook groups and maintaining a massive Excel sheet.",
        commercial_potential: 8,
        saas_feasibility: "High. A targeted aggregator with verified pet policies.",
        source_url: "https://dcard.tw/f/example",
      },
      {
        id: 2,
        platform: "Reddit",
        core_pain_point: "Freelancers struggle to calculate estimated quarterly taxes across different state lines.",
        current_workaround: "Paying a CPA $500/year or using complex IRS worksheets manually.",
        commercial_potential: 9,
        saas_feasibility: "High. Plaid integration + tax formula calculator.",
        source_url: "https://reddit.com/r/example",
      }
    ]);
  }, []);

  return (
    <main className="min-h-screen text-slate-200 p-8 font-sans max-w-6xl mx-auto">
      <header className="flex items-center justify-between mb-10 border-b border-slate-700 pb-6">
        <div>
          <h1 className="text-4xl font-extrabold tracking-tight text-white mb-2 flex items-center gap-3">
            <Activity className="text-emerald-400" size={32} />
            The Idea Engine
          </h1>
          <p className="text-slate-400 text-lg">Real-time SaaS opportunity discovery stream.</p>
        </div>
        <div className="flex items-center gap-2 bg-slate-800 px-4 py-2 rounded-full border border-slate-700">
          <span className="relative flex h-3 w-3">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
          </span>
          <span className="text-sm font-medium text-slate-300">Live Stream Active</span>
        </div>
      </header>

      <div className="grid gap-6">
        {insights.map((insight) => (
          <div key={insight.id} className="bg-slate-800 rounded-xl p-6 border border-slate-700 hover:border-emerald-500/50 transition-all shadow-lg">
            <div className="flex justify-between items-start mb-4">
              <span className="px-3 py-1 bg-slate-700 text-slate-300 rounded-md text-xs font-semibold uppercase tracking-wider">
                {insight.platform}
              </span>
              <div className="flex items-center gap-1 text-amber-400 font-bold bg-amber-400/10 px-3 py-1 rounded-full text-sm">
                <Flame size={16} /> Score: {insight.commercial_potential}/10
              </div>
            </div>
            
            <h2 className="text-xl font-bold text-white mb-3">"{insight.core_pain_point}"</h2>
            
            <div className="mb-4 text-slate-400 bg-slate-900/50 p-4 rounded-lg border border-slate-800">
              <strong className="text-slate-300 block mb-1 flex items-center gap-2"><Code2 size={16}/> Workaround:</strong> 
              {insight.current_workaround}
            </div>
            
            <div className="flex justify-between items-center text-sm">
              <span className="text-emerald-400 font-medium">SaaS Feasibility: {insight.saas_feasibility}</span>
              <a href={insight.source_url} target="_blank" className="text-blue-400 hover:text-blue-300 flex items-center gap-1 transition-colors">Source <ExternalLink size={14} /></a>
            </div>
          </div>
        ))}
      </div>
    </main>
  );
}