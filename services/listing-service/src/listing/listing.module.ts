import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { CategoryModule } from '../category/category.module';
import { AdminListingsController } from './admin-listings.controller';
import { ListingController } from './listing.controller';
import { ListingEventsConsumer } from './listing-events.consumer';
import { ListingService } from './listing.service';
import { Listing } from './entities/listing.entity';

@Module({
  imports: [TypeOrmModule.forFeature([Listing]), CategoryModule],
  controllers: [ListingController, AdminListingsController],
  providers: [ListingService, ListingEventsConsumer],
})
export class ListingModule {}
