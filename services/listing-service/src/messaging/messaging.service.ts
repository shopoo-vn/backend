import { randomUUID } from 'node:crypto';
import {
  Injectable,
  Logger,
  OnModuleDestroy,
  OnModuleInit,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as amqp from 'amqplib';

export type EventHandler = (type: string, data: unknown, eventId: string) => Promise<void>;

// Single connection + channel to RabbitMQ. Publishes the project-standard
// envelope to the topic exchange; supports idempotent consumers.
@Injectable()
export class MessagingService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(MessagingService.name);
  // amqplib 0.10.4+ : connect() resolves to a ChannelModel (createChannel/close live here).
  private connection?: Awaited<ReturnType<typeof amqp.connect>>;
  private channel?: amqp.Channel;
  private readonly exchange: string;
  private readonly url: string;

  constructor(config: ConfigService) {
    this.url = config.getOrThrow<string>('rabbitmq.url');
    this.exchange = config.getOrThrow<string>('rabbitmq.exchange');
  }

  async onModuleInit(): Promise<void> {
    const connection = await amqp.connect(this.url);
    const channel = await connection.createChannel();
    await channel.assertExchange(this.exchange, 'topic', { durable: true });
    this.connection = connection;
    this.channel = channel;
    this.logger.log(`connected to RabbitMQ, exchange="${this.exchange}"`);
  }

  async onModuleDestroy(): Promise<void> {
    try {
      await this.channel?.close();
      await this.connection?.close();
    } catch (err) {
      this.logger.warn(`error closing RabbitMQ: ${(err as Error).message}`);
    }
  }

  /** Publish an event with the standard envelope. Routing key === type. */
  publish(type: string, data: unknown): void {
    if (!this.channel) {
      this.logger.error(`cannot publish "${type}": channel not ready`);
      return;
    }
    const envelope = {
      eventId: randomUUID(),
      type,
      occurredAt: new Date().toISOString(),
      data,
    };
    this.channel.publish(this.exchange, type, Buffer.from(JSON.stringify(envelope)), {
      persistent: true,
      contentType: 'application/json',
    });
  }

  /** Bind a durable queue to routing keys and process messages idempotently. */
  async consume(queue: string, routingKeys: string[], handler: EventHandler): Promise<void> {
    if (!this.channel) {
      throw new Error('RabbitMQ channel not ready');
    }
    const channel = this.channel;
    await channel.assertQueue(queue, { durable: true });
    await Promise.all(routingKeys.map((key) => channel.bindQueue(queue, this.exchange, key)));
    await channel.consume(queue, (msg) => {
      if (!msg) {
        return;
      }
      void this.handleMessage(channel, msg, handler);
    });
    this.logger.log(`consuming "${queue}" bound to [${routingKeys.join(', ')}]`);
  }

  private async handleMessage(
    channel: amqp.Channel,
    msg: amqp.ConsumeMessage,
    handler: EventHandler,
  ): Promise<void> {
    try {
      const env = JSON.parse(msg.content.toString()) as {
        type: string;
        data: unknown;
        eventId: string;
      };
      await handler(env.type, env.data, env.eventId);
      channel.ack(msg);
    } catch (err) {
      this.logger.error(`failed to process message: ${(err as Error).message}`);
      // Don't requeue forever; drop to (a future) dead-letter queue.
      channel.nack(msg, false, false);
    }
  }
}
