import { config as loadEnv } from 'dotenv';
import { z } from 'zod';

loadEnv();

// Validated against process.env at boot — the app refuses to start on bad config.
const envSchema = z.object({
  CHAT_HTTP_PORT: z.coerce.number().int().positive().default(8006),
  CHAT_DB_URL: z
    .string()
    .refine((v) => /^postgres(ql)?:\/\//.test(v), 'must be a postgres:// url'),
  CHAT_REDIS_URL: z
    .string()
    .refine((v) => /^rediss?:\/\//.test(v), 'must be a redis:// url'),
  CHAT_RABBITMQ_URL: z
    .string()
    .refine((v) => /^amqps?:\/\//.test(v), 'must be an amqp:// url'),
  RABBITMQ_EXCHANGE: z.string().default('marketplace.events'),
  CHAT_JWT_PUBLIC_KEY_PATH: z.string().min(1),
  CHAT_JWT_ISSUER: z.string().default('marketplace-auth'),
  CHAT_CORS_ORIGINS: z.string().default('*'),
});

const parsed = envSchema.safeParse(process.env);
if (!parsed.success) {
  const issues = parsed.error.issues
    .map((i) => `${i.path.join('.')}: ${i.message}`)
    .join('; ');
  // eslint-disable-next-line no-console
  console.error(JSON.stringify({ level: 'fatal', service: 'chat-service', msg: `invalid config: ${issues}` }));
  process.exit(1);
}

const env = parsed.data;

export interface AppConfig {
  http: { port: number };
  db: { url: string; schema: string };
  redis: { url: string };
  rabbitmq: { url: string; exchange: string };
  jwt: { publicKeyPath: string; issuer: string };
  cors: { origins: string[] };
}

export const config: AppConfig = {
  http: { port: env.CHAT_HTTP_PORT },
  db: { url: env.CHAT_DB_URL, schema: 'chat' },
  redis: { url: env.CHAT_REDIS_URL },
  rabbitmq: { url: env.CHAT_RABBITMQ_URL, exchange: env.RABBITMQ_EXCHANGE },
  jwt: { publicKeyPath: env.CHAT_JWT_PUBLIC_KEY_PATH, issuer: env.CHAT_JWT_ISSUER },
  cors: { origins: env.CHAT_CORS_ORIGINS.split(',').map((s) => s.trim()) },
};
