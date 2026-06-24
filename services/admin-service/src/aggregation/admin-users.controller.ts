import {
  Controller,
  Get,
  Headers,
  Param,
  ParseUUIDPipe,
  Post,
  Query,
  UseGuards,
} from '@nestjs/common';
import { JwtAuthGuard } from '../common/auth/jwt-auth.guard';
import { Roles } from '../common/auth/roles.decorator';
import { RolesGuard } from '../common/auth/roles.guard';
import { PaginationQueryDto } from '../common/dto/pagination-query.dto';
import { UpstreamService } from '../upstream/upstream.service';

// Aggregates user data from the Auth Service (which owns the user records).
@Controller('admin/users')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles('admin')
export class AdminUsersController {
  constructor(private readonly upstream: UpstreamService) {}

  @Get()
  list(@Query() q: PaginationQueryDto, @Headers('authorization') auth?: string) {
    return this.upstream.authGet(`/users?page=${q.page}&limit=${q.limit}`, auth);
  }

  @Post(':id/ban')
  ban(@Param('id', ParseUUIDPipe) id: string, @Headers('authorization') auth?: string) {
    return this.upstream.authPatch(`/users/${id}/status`, { status: 'banned' }, auth);
  }

  @Post(':id/unban')
  unban(@Param('id', ParseUUIDPipe) id: string, @Headers('authorization') auth?: string) {
    return this.upstream.authPatch(`/users/${id}/status`, { status: 'active' }, auth);
  }
}
