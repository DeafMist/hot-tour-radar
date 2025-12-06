# Scraper

Python service powering scraper for hot tour channels in real time.

---

## Module Structure

- [parsers](./src/parsers) - parsers for channels in social networks and messengers
- [producers](./src/producers) - kafka producers for parsed data

---

## Kafka Schema

- **id** - internal id (uuid format)
- **source** - source of message (for example, `telegram`)
- **channel_id** - id of channel
- **channel_name** - channel name (string after `@` in telegram)
- **channel_title** - channel title
- **message_id** - channel message id
- **text** - unprocessed message text
- **date** - message date in [ISO-8601](https://en.wikipedia.org/wiki/ISO_8601) format

---

## Running locally

```bash
docker-compose up zookeeper kafka kafka-ui news-scraper --build
```
