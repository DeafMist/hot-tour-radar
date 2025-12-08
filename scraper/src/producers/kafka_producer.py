import json
import logging
import asyncio
from typing import Dict, Any, Optional
from kafka import KafkaProducer
from kafka.errors import KafkaError
from concurrent.futures import ThreadPoolExecutor

logger = logging.getLogger(__name__)


class KafkaNewsProducer:
    """Kafka Producer for sending hot tour news"""

    def __init__(self, bootstrap_servers: str, topic: str = "news_raw"):
        """
        Args:
            bootstrap_servers: kafka brokers addresses (separated by comma)
            topic: kafka topic name
        """
        self.topic = topic
        self.producer = self._init_producer(bootstrap_servers)
        self.executor = ThreadPoolExecutor(max_workers=2)
        logger.info(f"Kafka producer initialized for topic '{topic}'")

    def _init_producer(self, bootstrap_servers: str) -> KafkaProducer:
        """Initialize Kafka Producer"""
        try:
            return KafkaProducer(
                bootstrap_servers=bootstrap_servers.split(','),
                value_serializer=lambda v: json.dumps(v, ensure_ascii=False).encode('utf-8'),
                key_serializer=lambda k: str(k).encode('utf-8') if k else None,
                acks='all',
                retries=3,
                linger_ms=10,
                batch_size=16384,
                max_block_ms=5000,
                request_timeout_ms=30000
            )
        except Exception as e:
            logger.error(f"Failed to initialize Kafka producer: {e}")
            raise

    def send_news(self, news_data: Dict[str, Any], key: Optional[str] = None) -> bool:
        """Sync sending news to Kafka"""
        try:
            future = self.producer.send(
                topic=self.topic,
                key=key,
                value=news_data
            )

            # Blocking wait for confirmation
            record_metadata = future.get(timeout=10)

            logger.debug(
                f"News sent to Kafka - "
                f"topic: {record_metadata.topic}, "
                f"partition: {record_metadata.partition}, "
                f"offset: {record_metadata.offset}"
            )

            return True

        except KafkaError as e:
            logger.error(f"Kafka send error: {e}")
            return False
        except Exception as e:
            logger.error(f"Unexpected error sending to Kafka: {e}")
            return False

    async def send_news_async(self, news_data: Dict[str, Any], key: Optional[str] = None) -> bool:
        """Async sending news to Kafka"""
        loop = asyncio.get_running_loop()
        try:
            return await loop.run_in_executor(
                self.executor,
                lambda: self.send_news(news_data, key)
            )
        except Exception as e:
            logger.error(f"Error in async send: {e}")
            return False

    def flush(self):
        """Force sending news to Kafka"""
        self.producer.flush()

    def close(self):
        """Close resources"""
        self.producer.close(timeout=5)
        self.executor.shutdown(wait=True)
        logger.info("Kafka producer closed")
