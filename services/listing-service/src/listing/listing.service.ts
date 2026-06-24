import {
  ForbiddenException,
  Injectable,
  Logger,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { AuthUser } from '../common/auth/jwt.strategy';
import { Paginated } from '../common/interfaces/paginated.interface';
import { CategoryService } from '../category/category.service';
import { MessagingService } from '../messaging/messaging.service';
import { CreateListingDto } from './dto/create-listing.dto';
import { QueryListingsDto } from './dto/query-listings.dto';
import { UpdateListingDto } from './dto/update-listing.dto';
import { Listing, ListingStatus } from './entities/listing.entity';

@Injectable()
export class ListingService {
  private readonly logger = new Logger(ListingService.name);

  constructor(
    @InjectRepository(Listing)
    private readonly repo: Repository<Listing>,
    private readonly categories: CategoryService,
    private readonly messaging: MessagingService,
  ) {}

  async create(sellerId: string, dto: CreateListingDto): Promise<Listing> {
    await this.categories.findById(dto.categoryId); // 404s if missing

    const listing = this.repo.create({
      sellerId,
      title: dto.title,
      description: dto.description ?? '',
      price: dto.price,
      categoryId: dto.categoryId,
      location: dto.location ?? null,
      condition: dto.condition,
      mediaIds: dto.mediaIds ?? [],
      status: 'pending',
    });
    const saved = await this.repo.save(listing);

    // Publish AFTER the row is committed so Admin can queue it for moderation.
    this.messaging.publish('listing.created', {
      listingId: saved.id,
      sellerId: saved.sellerId,
      title: saved.title,
      price: saved.price,
      categoryId: saved.categoryId,
    });
    return saved;
  }

  async findById(id: string): Promise<Listing> {
    const listing = await this.repo.findOne({ where: { id } });
    if (!listing) {
      throw new NotFoundException('listing not found');
    }
    return listing;
  }

  async search(query: QueryListingsDto): Promise<Paginated<Listing>> {
    const { page, limit } = query;
    const qb = this.repo.createQueryBuilder('l').where('l.status = :status', { status: 'active' });

    if (query.q) {
      qb.andWhere(`l.search_vector @@ plainto_tsquery('simple', :q)`, { q: query.q });
    }
    if (query.categoryId) {
      qb.andWhere('l.category_id = :categoryId', { categoryId: query.categoryId });
    }
    if (query.minPrice !== undefined) {
      qb.andWhere('l.price >= :minPrice', { minPrice: query.minPrice });
    }
    if (query.maxPrice !== undefined) {
      qb.andWhere('l.price <= :maxPrice', { maxPrice: query.maxPrice });
    }
    if (query.location) {
      qb.andWhere('l.location ILIKE :location', { location: `%${query.location}%` });
    }
    if (query.condition) {
      qb.andWhere('l.condition = :condition', { condition: query.condition });
    }

    switch (query.sort) {
      case 'price_asc':
        qb.orderBy('l.price', 'ASC');
        break;
      case 'price_desc':
        qb.orderBy('l.price', 'DESC');
        break;
      default:
        qb.orderBy('l.created_at', 'DESC');
    }

    qb.skip((page - 1) * limit).take(limit);
    const [items, total] = await qb.getManyAndCount();
    return { items, page, limit, total };
  }

  async update(id: string, user: AuthUser, dto: UpdateListingDto): Promise<Listing> {
    const listing = await this.findById(id);
    this.assertCanMutate(listing, user);

    if (dto.categoryId && dto.categoryId !== listing.categoryId) {
      await this.categories.findById(dto.categoryId);
    }
    Object.assign(listing, {
      ...(dto.title !== undefined && { title: dto.title }),
      ...(dto.description !== undefined && { description: dto.description }),
      ...(dto.price !== undefined && { price: dto.price }),
      ...(dto.categoryId !== undefined && { categoryId: dto.categoryId }),
      ...(dto.location !== undefined && { location: dto.location }),
      ...(dto.condition !== undefined && { condition: dto.condition }),
      ...(dto.mediaIds !== undefined && { mediaIds: dto.mediaIds }),
    });
    const saved = await this.repo.save(listing);
    this.messaging.publish('listing.updated', { listingId: saved.id, sellerId: saved.sellerId });
    return saved;
  }

  async remove(id: string, user: AuthUser): Promise<void> {
    const listing = await this.findById(id);
    this.assertCanMutate(listing, user);
    await this.repo.delete({ id });
    this.messaging.publish('listing.deleted', { listingId: id, sellerId: listing.sellerId });
  }

  /** Applied from moderation events; idempotent (only moves a pending listing). */
  async applyModeration(listingId: string, status: ListingStatus): Promise<void> {
    const result = await this.repo.update(
      { id: listingId, status: 'pending' },
      { status },
    );
    if (result.affected === 0) {
      this.logger.debug(`moderation no-op for ${listingId} -> ${status} (already handled?)`);
    }
  }

  private assertCanMutate(listing: Listing, user: AuthUser): void {
    if (listing.sellerId !== user.userId && user.role !== 'admin') {
      throw new ForbiddenException('you do not own this listing');
    }
  }
}
