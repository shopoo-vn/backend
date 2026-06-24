import { readFileSync } from 'node:fs';
import { Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { PassportStrategy } from '@nestjs/passport';
import { ExtractJwt, Strategy } from 'passport-jwt';

export interface AuthUser {
  userId: string;
  role: string;
}

interface JwtPayload {
  sub: string;
  role: string;
  iss: string;
}

// Verifies access tokens with the Auth Service's PUBLIC key only (RS256).
@Injectable()
export class JwtStrategy extends PassportStrategy(Strategy) {
  constructor(config: ConfigService) {
    const publicKeyPath = config.get<string>('jwt.publicKeyPath');
    if (!publicKeyPath) {
      throw new Error('jwt.publicKeyPath is not configured');
    }
    super({
      jwtFromRequest: ExtractJwt.fromAuthHeaderAsBearerToken(),
      ignoreExpiration: false,
      secretOrKey: readFileSync(publicKeyPath, 'utf8'),
      algorithms: ['RS256'],
      issuer: config.get<string>('jwt.issuer'),
    });
  }

  validate(payload: JwtPayload): AuthUser {
    if (!payload?.sub) {
      throw new UnauthorizedException('invalid token');
    }
    return { userId: payload.sub, role: payload.role };
  }
}
