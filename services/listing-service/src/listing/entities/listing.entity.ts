import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';
import { numericTransformer } from '../../common/transformers/numeric.transformer';

export type ListingCondition = 'new' | 'like_new' | 'used';
export type ListingStatus = 'pending' | 'active' | 'rejected' | 'sold' | 'hidden';

@Entity({ name: 'listings' })
export class Listing {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Index()
  @Column({ name: 'seller_id', type: 'uuid' })
  sellerId: string;

  @Column()
  title: string;

  @Column({ type: 'text', default: '' })
  description: string;

  @Column({ type: 'numeric', precision: 14, scale: 0, transformer: numericTransformer })
  price: number;

  @Column({ name: 'category_id', type: 'uuid' })
  categoryId: string;

  @Column({ type: 'text', nullable: true })
  location: string | null;

  @Column({ type: 'text', default: 'used' })
  condition: ListingCondition;

  @Column({ type: 'text', default: 'pending' })
  status: ListingStatus;

  @Column({ name: 'media_ids', type: 'text', array: true, default: () => "'{}'" })
  mediaIds: string[];

  @CreateDateColumn({ name: 'created_at' })
  createdAt: Date;

  @UpdateDateColumn({ name: 'updated_at' })
  updatedAt: Date;
}
