import { Injectable } from '@nestjs/common';
import { ModerationService } from '../moderation/moderation.service';
import { ReportsService } from '../reports/reports.service';

export interface DashboardCounts {
  pending: number;
  approved: number;
  rejected: number;
  reportsOpen: number;
}

@Injectable()
export class DashboardService {
  constructor(
    private readonly moderation: ModerationService,
    private readonly reports: ReportsService,
  ) {}

  async counts(): Promise<DashboardCounts> {
    const [moderation, reportsOpen] = await Promise.all([
      this.moderation.counts(),
      this.reports.countOpen(),
    ]);
    return { ...moderation, reportsOpen };
  }
}
