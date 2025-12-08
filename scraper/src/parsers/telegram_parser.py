import logging
import uuid
from typing import Dict, Any, Optional
from telethon import TelegramClient, events
from telethon.tl.types import Message

logger = logging.getLogger(__name__)


class TelegramNewsParser:
    """Scraper for hot tours from Telegram"""

    def __init__(self, api_id: str, api_hash: str, producer, monitored_channels: list[str] | None = None):
        """
        Args:
            api_id: Telegram API ID
            api_hash: Telegram API Hash
            producer: KafkaNewsProducer
            monitored_channels: monitoring channels
        """
        try:
            self.api_id = int(api_id)
        except ValueError:
            raise ValueError(f"Invalid TELEGRAM_API_ID: '{api_id}' is not a valid integer") from None
        self.api_hash = api_hash
        self.producer = producer
        self.monitored_channels = monitored_channels or []
        self.monitored_channel_ids = set()
        self.client = TelegramClient('news_parser_session', self.api_id, self.api_hash)
        self._setup_handlers()

    def _setup_handlers(self):
        """Setting up handlers"""

        @self.client.on(events.NewMessage)
        async def message_handler(event):
            """Process new messages"""
            try:
                if await self._should_process_message(event.message):
                    news_data = await self._parse_message(event.message)
                    if news_data:
                        # Send by producer
                        key = str(news_data.get('channel_id', 'unknown'))
                        await self.producer.send_news_async(news_data, key=key)

                        logger.info(
                            f"Parsed news from {news_data.get('channel_name')}: {news_data.get('text', '')[:10]}...")

            except Exception as e:
                logger.error(f"Error in message handler: {e}")

    async def _should_process_message(self, message: Message) -> bool:
        """Check if message should be processed"""

        # Check if the message from right channel
        if not hasattr(message.peer_id, 'channel_id') and not hasattr(message.peer_id, 'chat_id'):
            return False

        # if channels is empty, do not process message
        if not self.monitored_channels:
            return False

        # Get current channel id
        current_channel_id = getattr(message.peer_id, 'channel_id', getattr(message.peer_id, 'chat_id', None))
        if not current_channel_id:
            return False

        # Find it among subscriptions
        if current_channel_id not in self.monitored_channel_ids:
            return False

        return True

    async def _parse_message(self, message: Message) -> Optional[Dict[str, Any]]:
        """Parse message and build kafka message"""
        try:
            chat = await message.get_chat()

            channel_id = getattr(message.peer_id, 'channel_id', getattr(message.peer_id, 'chat_id', None))
            news_data = {
                "id": f"{uuid.uuid4()}",
                "source": "telegram",
                "channel_id": channel_id,
                "channel_name": chat.username if chat else None,
                "channel_title": chat.title if chat else None,
                "message_id": message.id,
                "text": message.text or message.message or "",
                "date": message.date.isoformat() if message.date else None,
            }

            return news_data

        except Exception as e:
            logger.error(f"Error parsing message: {e}")
            return None

    async def add_channel(self, channel_identifier: str):
        """Add channel for monitoring"""
        try:
            entity = await self.client.get_entity(channel_identifier)

            # Add to cache
            self.monitored_channel_ids.add(entity.id)

            logger.info(f"Added channel to monitoring: {entity.username or entity.title}")
            return True

        except Exception as e:
            logger.error(f"Failed to add channel {channel_identifier}: {e}")
            return False

    async def start(self):
        """Start scraper"""
        try:
            await self.client.start()
            logger.info("Telegram parser started")

            # Add channels from config
            for channel in self.monitored_channels:
                if isinstance(channel, str):
                    await self.add_channel(channel)

            await self.client.run_until_disconnected()

        except KeyboardInterrupt:
            logger.info("Parser stopped by user")
        except Exception as e:
            logger.error(f"Parser error: {e}")
            raise
        finally:
            await self.client.disconnect()
            self.producer.close()
