import { Injectable, Logger, OnApplicationBootstrap } from '@nestjs/common';
import { MessagingService } from '../messaging/messaging.service';
import { ListingService } from './listing.service';

interface ModerationPayload {
  listingId: string;
}

// Listens for moderation decisions from Admin Service and flips listing status.
// Subscribes in onApplicationBootstrap (not onModuleInit) so MessagingService's
// onModuleInit — which opens the RabbitMQ channel — is guaranteed to have run.
@Injectable()
export class ListingEventsConsumer implements OnApplicationBootstrap {
  private readonly logger = new Logger(ListingEventsConsumer.name);

  constructor(
    private readonly messaging: MessagingService,
    private readonly listings: ListingService,
  ) {}

  async onApplicationBootstrap(): Promise<void> {
    await this.messaging.consume(
      'listing-service.moderation',
      ['listing.approved', 'listing.rejected'],
      async (type, data) => {
        const { listingId } = data as ModerationPayload;
        if (!listingId) {
          this.logger.warn(`${type} event missing listingId`);
          return;
        }
        const status = type === 'listing.approved' ? 'active' : 'rejected';
        await this.listings.applyModeration(listingId, status);
      },
    );
  }
}
