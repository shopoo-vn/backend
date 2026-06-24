import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { AggregationModule } from './aggregation/aggregation.module';
import { AuthModule } from './common/auth/auth.module';
import configuration from './config/configuration';
import { envValidationSchema } from './config/env.validation';
import { DashboardModule } from './dashboard/dashboard.module';
import { dataSourceOptions } from './database/data-source';
import { HealthController } from './health.controller';
import { MessagingModule } from './messaging/messaging.module';
import { ModerationModule } from './moderation/moderation.module';
import { ReportsModule } from './reports/reports.module';
import { UpstreamModule } from './upstream/upstream.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [configuration],
      validationSchema: envValidationSchema,
    }),
    TypeOrmModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (config: ConfigService) => ({
        ...dataSourceOptions,
        url: config.getOrThrow<string>('database.url'),
        migrationsRun: true,
      }),
    }),
    AuthModule,
    MessagingModule,
    UpstreamModule,
    ModerationModule,
    ReportsModule,
    DashboardModule,
    AggregationModule,
  ],
  controllers: [HealthController],
})
export class AppModule {}
