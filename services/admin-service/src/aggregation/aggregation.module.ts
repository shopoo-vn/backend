import { Module } from '@nestjs/common';
import { AdminListingsController } from './admin-listings.controller';
import { AdminUsersController } from './admin-users.controller';

// Read/aggregate endpoints that proxy to Auth + Listing (UpstreamService is global).
@Module({
  controllers: [AdminUsersController, AdminListingsController],
})
export class AggregationModule {}
