import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { AuthModule } from './common/auth/auth.module';
import { CategoryModule } from './category/category.module';
import configuration from './config/configuration';
import { envValidationSchema } from './config/env.validation';
import { dataSourceOptions } from './database/data-source';
import { HealthController } from './health.controller';
import { ListingModule } from './listing/listing.module';
import { MessagingModule } from './messaging/messaging.module';

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
    CategoryModule,
    ListingModule,
  ],
  controllers: [HealthController],
})
export class AppModule {}
