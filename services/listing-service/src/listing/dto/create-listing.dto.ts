import {
  ArrayMaxSize,
  IsArray,
  IsIn,
  IsInt,
  IsOptional,
  IsString,
  IsUUID,
  Max,
  MaxLength,
  Min,
  MinLength,
} from 'class-validator';
import { ListingCondition } from '../entities/listing.entity';

export class CreateListingDto {
  @IsString()
  @MinLength(3)
  @MaxLength(200)
  title: string;

  @IsOptional()
  @IsString()
  @MaxLength(5000)
  description?: string;

  // Price in VND (integer đồng).
  @IsInt()
  @Min(0)
  @Max(100_000_000_000)
  price: number;

  @IsUUID()
  categoryId: string;

  @IsOptional()
  @IsString()
  @MaxLength(200)
  location?: string;

  @IsIn(['new', 'like_new', 'used'])
  condition: ListingCondition;

  @IsOptional()
  @IsArray()
  @ArrayMaxSize(12)
  @IsUUID('4', { each: true })
  mediaIds?: string[];
}
