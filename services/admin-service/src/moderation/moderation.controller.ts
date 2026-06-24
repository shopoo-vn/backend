import {
  Body,
  Controller,
  Get,
  Param,
  ParseUUIDPipe,
  Post,
  Query,
  UseGuards,
} from '@nestjs/common';
import { CurrentUser } from '../common/auth/current-user.decorator';
import { JwtAuthGuard } from '../common/auth/jwt-auth.guard';
import { AuthUser } from '../common/auth/jwt.strategy';
import { Roles } from '../common/auth/roles.decorator';
import { RolesGuard } from '../common/auth/roles.guard';
import { PaginationQueryDto } from '../common/dto/pagination-query.dto';
import { Paginated } from '../common/interfaces/paginated.interface';
import { RejectListingDto } from './dto/reject-listing.dto';
import { ModerationItem } from './entities/moderation-item.entity';
import { ModerationService } from './moderation.service';

@Controller('admin/moderation')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles('admin')
export class ModerationController {
  constructor(private readonly moderation: ModerationService) {}

  @Get('queue')
  queue(@Query() query: PaginationQueryDto): Promise<Paginated<ModerationItem>> {
    return this.moderation.queue(query.page, query.limit);
  }

  @Post(':listingId/approve')
  approve(
    @CurrentUser() user: AuthUser,
    @Param('listingId', ParseUUIDPipe) listingId: string,
  ): Promise<ModerationItem> {
    return this.moderation.approve(listingId, user.userId);
  }

  @Post(':listingId/reject')
  reject(
    @CurrentUser() user: AuthUser,
    @Param('listingId', ParseUUIDPipe) listingId: string,
    @Body() dto: RejectListingDto,
  ): Promise<ModerationItem> {
    return this.moderation.reject(listingId, user.userId, dto.reason);
  }
}
