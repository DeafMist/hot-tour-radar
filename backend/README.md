# Backend

Go services powering ingestion, search, and retention for the news stream.

## Services

- worker – Kafka consumer that cleans raw news payloads, enriches them with keywords, removes duplicates, and indexes the result into Elasticsearch.
- api – HTTP service that exposes news search, filtering, and aggregation endpoints backed by Elasticsearch.
- retention – Lightweight cron-style service that periodically deletes outdated news documents to keep the cluster lean.

## Shared schema

All internal services operate on the same canonical JSON shape: id, title, text, timestamp, keywords, source. The scraper publishes title, text, timestamp, and source to Kafka (`news_raw` topic). The worker populates id and keywords before indexing to Elasticsearch.

## Configuration

Each service is configured exclusively through environment variables:

- `KAFKA_BROKERS` – Comma-separated list of Kafka bootstrap servers. Default `kafka:9092`.
- `KAFKA_TOPIC` – Topic to consume or produce news messages. Default `news_raw`.
- `KAFKA_CONSUMER_GROUP` – Consumer group for the worker service. Default `news-worker`.
- `ELASTICSEARCH_ADDR` – Elasticsearch URL (http/https). Default `http://elasticsearch:9200`.
- `ELASTICSEARCH_INDEX` – Target index for news documents. Default `news`.
- `API_BIND_ADDR` – API listen address (`host:port`). Default `0.0.0.0:8080`.
- `RETENTION_CRON` – Interval spec (`1h`, `12h`, `24h`, …) for cleanup runs. Default `24h`.
- `RETENTION_MAX_AGE` – Maximum document age (Go duration) before deletion. Default `168h` (7 days).

All durations follow Go's duration syntax (e.g., `72h`, `15m`).

## Running locally

```bash
docker compose up --build api worker retention
```

The compose stack also provisions Zookeeper, Kafka, and Elasticsearch. Ensure the scraper publishes JSON payloads to the `news_raw` topic before starting the worker.

## API quickstart

```http
GET http://localhost:8080/news?q=турция&keywords=пляж,авиа&size=5
```

Optional query params:

- `q` – full-text search phrase (title + text)
- `keywords` – comma-separated keywords to filter on
- `source` – exact match on source field
- `from`/`size` – pagination controls (default 0/20)
- `sort` – `<field>:<direction>` (default `timestamp:desc`)
- `start`/`end` – RFC3339 timestamps limiting the range
