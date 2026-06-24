import { Injectable, Logger, OnApplicationBootstrap } from '@nestjs/common';
import { MessagingService } from '../messaging/messaging.service';
import { ModerationService } from './moderation.service';

interface ListingCreatedPayload {
  listingId: string;
  sellerId?: string;
  title?: string;
}

// Listens for newly created listings and queues them for moderation (pending).
// Subscribes in onApplicationBootstrap (not onModuleInit) so MessagingService's
// onModuleInit — which opens the RabbitMQ channel — is guaranteed to have run.
@Injectable()
export class ModerationEventsConsumer implements OnApplicationBootstrap {
  private readonly logger = new Logger(ModerationEventsConsumer.name);

  constructor(
    private readonly messaging: MessagingService,
    private readonly moderation: ModerationService,
  ) {}

  async onApplicationBootstrap(): Promise<void> {
    await this.messaging.consume(
      'admin-service.moderation',
      ['listing.created'],
      async (type, data) => {
        const payload = data as ListingCreatedPayload;
        if (!payload?.listingId) {
          this.logger.warn(`${type} event missing listingId`);
          return;
        }
        await this.moderation.upsertFromListingCreated(payload);
      },
    );
  }
}
