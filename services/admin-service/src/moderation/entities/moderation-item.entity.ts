import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  PrimaryColumn,
  UpdateDateColumn,
} from 'typeorm';

export type ModerationStatus = 'pending' | 'approved' | 'rejected';

@Entity({ name: 'moderation_items' })
export class ModerationItem {
  // listing_id is the primary key — one moderation row per listing.
  @PrimaryColumn({ name: 'listing_id', type: 'uuid' })
  listingId: string;

  @Column({ name: 'seller_id', type: 'uuid', nullable: true })
  sellerId: string | null;

  @Column({ type: 'text', nullable: true })
  title: string | null;

  @Index()
  @Column({ type: 'text', default: 'pending' })
  status: ModerationStatus;

  @Column({ name: 'reviewed_by', type: 'uuid', nullable: true })
  reviewedBy: string | null;

  @Column({ type: 'text', nullable: true })
  reason: string | null;

  @CreateDateColumn({ name: 'created_at' })
  createdAt: Date;

  @UpdateDateColumn({ name: 'updated_at' })
  updatedAt: Date;
}
