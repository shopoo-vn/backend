import * as Joi from 'joi';

// Validated against process.env at boot — the app refuses to start on bad config.
export const envValidationSchema = Joi.object({
  ADMIN_HTTP_PORT: Joi.number().port().default(8005),
  ADMIN_DB_URL: Joi.string().uri({ scheme: ['postgres', 'postgresql'] }).required(),
  ADMIN_RABBITMQ_URL: Joi.string().uri({ scheme: ['amqp', 'amqps'] }).required(),
  RABBITMQ_EXCHANGE: Joi.string().default('marketplace.events'),
  ADMIN_JWT_PUBLIC_KEY_PATH: Joi.string().required(),
  ADMIN_JWT_ISSUER: Joi.string().default('marketplace-auth'),
  ADMIN_LISTING_BASE_URL: Joi.string().uri().required(),
  ADMIN_CORS_ORIGINS: Joi.string().default('*'),
});
