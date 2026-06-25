import { Controller, Get, Headers, Req, UseGuards } from '@nestjs/common';
import type { Request } from 'express';
import { JwtAuthGuard } from '../common/auth/jwt-auth.guard';
import { Roles } from '../common/auth/roles.decorator';
import { RolesGuard } from '../common/auth/roles.guard';
import { UpstreamService } from '../upstream/upstream.service';

// Aggregates listing data from the Listing Service (every status, admin view).
@Controller('admin/listings')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles('admin')
export class AdminListingsController {
  constructor(private readonly upstream: UpstreamService) {}

  @Get()
  list(@Req() req: Request, @Headers('authorization') auth?: string) {
    // Forward the raw query string (page, limit, status, q, …) to Listing Service,
    // which validates it with its AdminQueryListingsDto.
    const qIndex = req.url.indexOf('?');
    const qs = qIndex >= 0 ? req.url.slice(qIndex) : '';
    return this.upstream.listingGet(`/admin/listings${qs}`, auth);
  }
}
