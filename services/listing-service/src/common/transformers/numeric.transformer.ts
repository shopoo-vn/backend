import { ValueTransformer } from 'typeorm';

// Postgres numeric/bigint comes back as a string in node-pg. Map it to a JS number.
export const numericTransformer: ValueTransformer = {
  to: (value?: number | null) => value,
  from: (value?: string | null): number | null =>
    value === null || value === undefined ? null : Number(value),
};
