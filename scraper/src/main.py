import asyncio
import logging
import os
from dotenv import load_dotenv

from src.parsers.telegram_parser import TelegramNewsParser
from src.producers.kafka_producer import KafkaNewsProducer

# Logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def load_config() -> dict:
    """Load config from .env"""
    load_dotenv()

    return {
        # Telegram
        "telegram_api_id": os.getenv("TELEGRAM_API_ID"),
        "telegram_api_hash": os.getenv("TELEGRAM_API_HASH"),

        # Kafka
        "kafka_brokers": os.getenv("KAFKA_BROKERS"),
        "kafka_topic": os.getenv("KAFKA_TOPIC", "news_raw"),

        # Channels for monitoring separated by comma
        "channels": [c.strip() for c in os.getenv("TELEGRAM_CHANNELS", "").split(',') if c.strip()],
    }


async def main():
    """Main function"""
    config = load_config()

    # Check required variables
    if not config["telegram_api_id"] or not config["telegram_api_hash"] or not config["kafka_brokers"]:
        logger.error("TELEGRAM_API_ID, TELEGRAM_API_HASH and KAFKA_BROKERS must be set in .env")
        return

    # Initialize producer
    logger.info(f"Initializing Kafka producer for {config['kafka_brokers']}")
    producer = KafkaNewsProducer(
        bootstrap_servers=config["kafka_brokers"],
        topic=config["kafka_topic"]
    )

    # Initialize parser
    parser = TelegramNewsParser(
        api_id=config["telegram_api_id"],
        api_hash=config["telegram_api_hash"],
        producer=producer,
        monitored_channels=config["channels"]
    )

    # Start parser
    logger.info(f"Starting parser with {len(config['channels'])} channels")
    try:
        await parser.start()
    except Exception as e:
        logger.error(f"Failed to start parser: {e}")
    finally:
        logger.info("Parser stopped")


if __name__ == "__main__":
    asyncio.run(main())
