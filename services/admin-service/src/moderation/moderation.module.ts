import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ModerationController } from './moderation.controller';
import { ModerationEventsConsumer } from './moderation-events.consumer';
import { ModerationService } from './moderation.service';
import { ModerationItem } from './entities/moderation-item.entity';

@Module({
  imports: [TypeOrmModule.forFeature([ModerationItem])],
  controllers: [ModerationController],
  providers: [ModerationService, ModerationEventsConsumer],
  exports: [ModerationService],
})
export class ModerationModule {}
