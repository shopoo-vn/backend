import 'reflect-metadata';
import { config as loadEnv } from 'dotenv';
import { DataSource, DataSourceOptions } from 'typeorm';
import { Category } from '../category/category.entity';
import { Listing } from '../listing/entities/listing.entity';

loadEnv();

// Shared by the TypeORM CLI (migrations) and the app (see app.module.ts).
export const dataSourceOptions: DataSourceOptions = {
  type: 'postgres',
  url: process.env.LISTING_DB_URL,
  schema: 'listing',
  entities: [Listing, Category],
  migrations: [__dirname + '/migrations/*.{ts,js}'],
  synchronize: false,
  logging: false,
};

export const AppDataSource = new DataSource(dataSourceOptions);
export default AppDataSource;
