# The Idea Engine Architecture

An automated platform that scrapes, analyzes, and streams user pain points from online communities (Reddit, Dcard, App Store, TikTok) to discover highly validated SaaS opportunities.

## System Architecture

### 1. Ingestion Layer (Go + Redis)
*   **Scrapers:** Lightweight Go workers that fetch data via APIs (Reddit, Dcard) and RSS (App Store).
*   **Deduplication:** Redis is used to store Post IDs with a TTL to ensure the same post is never processed twice.
*   **Queueing:** Scraped and deduplicated text is pushed to a **Kafka** topic for processing.

### 2. Intelligence Layer (Go + Gemini 1.5 Flash)
*   **Workers:** Go Goroutines consume the Kafka topic.
*   **LLM Analysis:** Prompts Gemini 1.5 Flash acting as a Senior SaaS PM to extract:
    *   Core Pain Points
    *   Current Workarounds
    *   Commercial Potential (1-10)
    *   SaaS Feasibility

### 3. Storage Layer (PostgreSQL)
*   **Relational Data:** Structured insights are stored in PostgreSQL.
*   **Vector Search:** `pgvector` will be used for similarity clustering to identify recurring global pain points.

### 4. Presentation Layer (Next.js)
*   **Tech Stack:** Next.js + Tailwind CSS.
*   **Real-time Streaming:** Server-Sent Events (SSE) or WebSockets feed a live "Pain Point Stream" to the dashboard.

## Commercialization Roadmap
*   **Phase 1:** Internal Tool (Personal SaaS Idea Validation).
*   **Phase 2:** Idea-as-a-Service (Curated Newsletter for Indie Hackers).
*   **Phase 3:** B2B Custom Monitoring (Competitor negative review tracking).

## Tech Stack Summary
*   **Backend:** Golang, Gin
*   **Frontend:** Next.js, React, Tailwind CSS
*   **Database/Cache:** PostgreSQL, pgvector, Redis
*   **Message Broker:** Apache Kafka
*   **AI/LLM:** Google Gemini 1.5 Flash
