import { IsIn, IsOptional } from 'class-validator';
import { QueryListingsDto } from './query-listings.dto';

// Admin listing query: same filters as the public search, plus an optional
// status filter (admins can see every status, not just "active").
export class AdminQueryListingsDto extends QueryListingsDto {
  @IsOptional()
  @IsIn(['pending', 'active', 'rejected', 'sold', 'hidden'])
  status?: string;
}
