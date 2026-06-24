import * as Joi from 'joi';

// Validated against process.env at boot — the app refuses to start on bad config.
export const envValidationSchema = Joi.object({
  LISTING_HTTP_PORT: Joi.number().port().default(8004),
  LISTING_DB_URL: Joi.string().uri({ scheme: ['postgres', 'postgresql'] }).required(),
  LISTING_RABBITMQ_URL: Joi.string().uri({ scheme: ['amqp', 'amqps'] }).required(),
  RABBITMQ_EXCHANGE: Joi.string().default('marketplace.events'),
  LISTING_JWT_PUBLIC_KEY_PATH: Joi.string().required(),
  LISTING_JWT_ISSUER: Joi.string().default('marketplace-auth'),
  LISTING_CORS_ORIGINS: Joi.string().default('*'),
});
