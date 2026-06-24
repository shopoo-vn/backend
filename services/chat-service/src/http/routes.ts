import { Router } from 'express';
import { requireAuth } from './auth.middleware';
import {
  createConversation,
  listConversations,
  listMessages,
} from './conversations.controller';

export const router = Router();

router.get('/conversations', requireAuth, listConversations);
router.post('/conversations', requireAuth, createConversation);
router.get('/conversations/:id/messages', requireAuth, listMessages);
