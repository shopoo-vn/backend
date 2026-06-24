import { randomUUID } from 'node:crypto';
import * as amqp from 'amqplib';
import { config } from '../config';
import { logger } from './logger';

// Single connection + channel to RabbitMQ. Publishes the project-standard
// envelope to the topic exchange "marketplace.events".
let connection: Awaited<ReturnType<typeof amqp.connect>> | undefined;
let channel: amqp.Channel | undefined;

export async function initRabbit(): Promise<void> {
  connection = await amqp.connect(config.rabbitmq.url);
  channel = await connection.createChannel();
  await channel.assertExchange(config.rabbitmq.exchange, 'topic', { durable: true });
  logger.info('connected to RabbitMQ', { exchange: config.rabbitmq.exchange });
}

/**
 * Publish an event with the standard envelope. Routing key === type. Never
 * throws into the request/socket path — a broker hiccup must not lose the
 * already-committed DB write.
 */
export function publishEvent(type: string, data: unknown): void {
  if (!channel) {
    logger.error('cannot publish: channel not ready', { type });
    return;
  }
  const envelope = {
    eventId: randomUUID(),
    type,
    occurredAt: new Date().toISOString(),
    data,
  };
  try {
    channel.publish(
      config.rabbitmq.exchange,
      type,
      Buffer.from(JSON.stringify(envelope)),
      { persistent: true, contentType: 'application/json' },
    );
  } catch (err) {
    logger.error('failed to publish event', { type, err: (err as Error).message });
  }
}

export async function closeRabbit(): Promise<void> {
  try {
    await channel?.close();
    await connection?.close();
  } catch (err) {
    logger.warn('error closing RabbitMQ', { err: (err as Error).message });
  }
}
