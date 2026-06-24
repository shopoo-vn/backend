export default () => ({
  http: {
    port: parseInt(process.env.ADMIN_HTTP_PORT ?? '8005', 10),
  },
  database: {
    url: process.env.ADMIN_DB_URL as string,
    schema: 'admin',
  },
  rabbitmq: {
    url: process.env.ADMIN_RABBITMQ_URL as string,
    exchange: process.env.RABBITMQ_EXCHANGE ?? 'marketplace.events',
  },
  jwt: {
    publicKeyPath: process.env.ADMIN_JWT_PUBLIC_KEY_PATH as string,
    issuer: process.env.ADMIN_JWT_ISSUER ?? 'marketplace-auth',
  },
  listing: {
    baseUrl: process.env.ADMIN_LISTING_BASE_URL as string,
  },
  auth: {
    baseUrl: process.env.ADMIN_AUTH_BASE_URL as string,
  },
  cors: {
    origins: (process.env.ADMIN_CORS_ORIGINS ?? '*').split(',').map((s) => s.trim()),
  },
});
