# Hot Tour Radar

---

## Configuration

This service needs `.env` file with following variables:

```dotenv
# Telegram API
TELEGRAM_API_ID=api_id
TELEGRAM_API_HASH=api_hash

# Telegram channel separated by comma
TELEGRAM_CHANNELS=@channel1,@channel2

# Kafka (for scraper local run)
KAFKA_BROKERS=localhost:9092
```

For getting `api_id` and `api_hash`, you can go to https://core.telegram.org/api/obtaining_api_id and follow instruction about creating your Telegram Application.

---

## Telegram Authorization

Before starting service, you need to get a telegram session for [telethon](https://github.com/LonamiWebs/Telethon).
For it, you need to run [main.py](./scraper/src/main.py) in IDEA for passing authorization.
Then put session file to [src](./scraper/src) folder with the name `news_parser_session.session`.

---

## Run application

```bash
make compose-up
```

---

## Kafka UI

Kafka UI starts at http://localhost:8081

## React application

React application starts at http://localhost:3000

---

## Additional Information

For additional information about all modules go to local README files

- [backend](./backend/README.md)
- [scraper](./scraper/README.md)
- [frontend](./frontend/README.md)
