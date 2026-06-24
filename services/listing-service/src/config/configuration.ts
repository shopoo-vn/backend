export default () => ({
  http: {
    port: parseInt(process.env.LISTING_HTTP_PORT ?? '8004', 10),
  },
  database: {
    url: process.env.LISTING_DB_URL as string,
    schema: 'listing',
  },
  rabbitmq: {
    url: process.env.LISTING_RABBITMQ_URL as string,
    exchange: process.env.RABBITMQ_EXCHANGE ?? 'marketplace.events',
  },
  jwt: {
    publicKeyPath: process.env.LISTING_JWT_PUBLIC_KEY_PATH as string,
    issuer: process.env.LISTING_JWT_ISSUER ?? 'marketplace-auth',
  },
  cors: {
    origins: (process.env.LISTING_CORS_ORIGINS ?? '*').split(',').map((s) => s.trim()),
  },
});
