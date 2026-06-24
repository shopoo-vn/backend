import { z } from 'zod';

export const createConversationSchema = z.object({
  sellerId: z.string().uuid(),
  listingId: z.string().uuid().nullish(),
});

export const listMessagesQuerySchema = z.object({
  before: z.string().datetime().optional(),
  limit: z.coerce.number().int().min(1).max(100).default(20),
});

export type CreateConversationBody = z.infer<typeof createConversationSchema>;
