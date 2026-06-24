import {
  Controller,
  Get,
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
import { Paginated } from '../common/interfaces/paginated.interface';
import { Report } from './entities/report.entity';
import { ReportsService } from './reports.service';

@Controller('admin/reports')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles('admin')
export class ReportsController {
  constructor(private readonly reports: ReportsService) {}

  @Get()
  list(@Query() query: PaginationQueryDto): Promise<Paginated<Report>> {
    return this.reports.list(query.page, query.limit);
  }

  @Post(':id/resolve')
  resolve(@Param('id', ParseUUIDPipe) id: string): Promise<Report> {
    return this.reports.resolve(id);
  }
}
