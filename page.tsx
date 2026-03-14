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
    const fetchInsights = async () => {
      try {
        const res = await fetch('http://localhost:8080/api/insights');
        const data = await res.json();
        setInsights(data || []);
      } catch (error) {
        console.error("Failed to fetch insights:", error);
      }
    };
    
    fetchInsights();
    const interval = setInterval(fetchInsights, 5000); // Poll every 5 seconds for new ideas
    return () => clearInterval(interval);
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
