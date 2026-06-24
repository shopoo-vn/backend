import 'reflect-metadata';
import { config as loadEnv } from 'dotenv';
import { DataSource, DataSourceOptions } from 'typeorm';
import { ModerationItem } from '../moderation/entities/moderation-item.entity';
import { Report } from '../reports/entities/report.entity';

loadEnv();

// Shared by the TypeORM CLI (migrations) and the app (see app.module.ts).
export const dataSourceOptions: DataSourceOptions = {
  type: 'postgres',
  url: process.env.ADMIN_DB_URL,
  schema: 'admin',
  entities: [ModerationItem, Report],
  migrations: [__dirname + '/migrations/*.{ts,js}'],
  synchronize: false,
  logging: false,
};

export const AppDataSource = new DataSource(dataSourceOptions);
export default AppDataSource;
