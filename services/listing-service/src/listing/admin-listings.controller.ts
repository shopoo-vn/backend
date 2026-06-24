import { Controller, Get, Query, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../common/auth/jwt-auth.guard';
import { Roles } from '../common/auth/roles.decorator';
import { RolesGuard } from '../common/auth/roles.guard';
import { Paginated } from '../common/interfaces/paginated.interface';
import { AdminQueryListingsDto } from './dto/admin-query-listings.dto';
import { Listing } from './entities/listing.entity';
import { ListingService } from './listing.service';

// Admin-only listing access: every status, not just active. Called by admin-service.
@Controller('admin/listings')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles('admin')
export class AdminListingsController {
  constructor(private readonly listings: ListingService) {}

  @Get()
  list(@Query() query: AdminQueryListingsDto): Promise<Paginated<Listing>> {
    return this.listings.adminSearch(query);
  }
}
