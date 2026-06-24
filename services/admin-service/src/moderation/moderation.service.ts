import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Paginated } from '../common/interfaces/paginated.interface';
import { MessagingService } from '../messaging/messaging.service';
import { ModerationItem } from './entities/moderation-item.entity';

interface ListingCreatedPayload {
  listingId: string;
  sellerId?: string;
  title?: string;
}

@Injectable()
export class ModerationService {
  private readonly logger = new Logger(ModerationService.name);

  constructor(
    @InjectRepository(ModerationItem)
    private readonly repo: Repository<ModerationItem>,
    private readonly messaging: MessagingService,
  ) {}

  /** Idempotent upsert from a listing.created event — queues the item as pending. */
  async upsertFromListingCreated(payload: ListingCreatedPayload): Promise<void> {
    await this.repo
      .createQueryBuilder()
      .insert()
      .into(ModerationItem)
      .values({
        listingId: payload.listingId,
        sellerId: payload.sellerId ?? null,
        title: payload.title ?? null,
        status: 'pending',
      })
      .orIgnore()
      .execute();
  }

  async queue(page: number, limit: number): Promise<Paginated<ModerationItem>> {
    const [items, total] = await this.repo.findAndCount({
      where: { status: 'pending' },
      order: { createdAt: 'DESC' },
      skip: (page - 1) * limit,
      take: limit,
    });
    return { items, page, limit, total };
  }

  async approve(listingId: string, adminId: string): Promise<ModerationItem> {
    const item = await this.findPending(listingId);
    item.status = 'approved';
    item.reviewedBy = adminId;
    item.reason = null;
    const saved = await this.repo.save(item);

    // Publish AFTER the row is committed: Listing flips status, Noti pushes to seller.
    this.messaging.publish('listing.approved', { listingId, sellerId: item.sellerId });
    return saved;
  }

  async reject(listingId: string, adminId: string, reason: string): Promise<ModerationItem> {
    const item = await this.findPending(listingId);
    item.status = 'rejected';
    item.reviewedBy = adminId;
    item.reason = reason;
    const saved = await this.repo.save(item);

    this.messaging.publish('listing.rejected', { listingId, reason, sellerId: item.sellerId });
    return saved;
  }

  private async findPending(listingId: string): Promise<ModerationItem> {
    const item = await this.repo.findOne({ where: { listingId } });
    if (!item) {
      throw new NotFoundException('moderation item not found');
    }
    return item;
  }

  async counts(): Promise<{ pending: number; approved: number; rejected: number }> {
    const rows = await this.repo
      .createQueryBuilder('m')
      .select('m.status', 'status')
      .addSelect('COUNT(*)', 'count')
      .groupBy('m.status')
      .getRawMany<{ status: string; count: string }>();

    const counts = { pending: 0, approved: 0, rejected: 0 };
    for (const row of rows) {
      if (row.status in counts) {
        counts[row.status as keyof typeof counts] = Number(row.count);
      }
    }
    return counts;
  }
}
