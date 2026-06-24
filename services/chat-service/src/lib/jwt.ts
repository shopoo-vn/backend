import { readFileSync } from 'node:fs';
import jwt from 'jsonwebtoken';
import { config } from '../config';

export interface AuthUser {
  userId: string;
  role: string;
}

interface AccessTokenClaims {
  sub?: string;
  role?: string;
  iss?: string;
}

// Load the Auth Service PUBLIC key once at startup. This service NEVER holds a
// private key — it only verifies RS256 access tokens signed by auth-service.
const publicKey = readFileSync(config.jwt.publicKeyPath, 'utf8');

/**
 * Verify an RS256 access token with the public key. Enforces alg=RS256, issuer
 * and expiry. Throws on any failure. Returns the authenticated user on success.
 */
export function verifyAccessToken(token: string): AuthUser {
  const decoded = jwt.verify(token, publicKey, {
    algorithms: ['RS256'],
    issuer: config.jwt.issuer,
  }) as AccessTokenClaims;

  if (!decoded.sub) {
    throw new Error('token missing sub');
  }
  return { userId: decoded.sub, role: decoded.role ?? 'user' };
}
