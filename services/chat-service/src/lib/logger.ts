// Minimal structured (JSON) logger — { service, level, msg, ...ctx }.
type Level = 'debug' | 'info' | 'warn' | 'error';

const SERVICE = 'chat-service';

function emit(level: Level, msg: string, ctx?: Record<string, unknown>): void {
  const line = JSON.stringify({
    service: SERVICE,
    level,
    msg,
    time: new Date().toISOString(),
    ...ctx,
  });
  // eslint-disable-next-line no-console
  if (level === 'error') console.error(line);
  else console.log(line);
}

export const logger = {
  debug: (msg: string, ctx?: Record<string, unknown>) => emit('debug', msg, ctx),
  info: (msg: string, ctx?: Record<string, unknown>) => emit('info', msg, ctx),
  warn: (msg: string, ctx?: Record<string, unknown>) => emit('warn', msg, ctx),
  error: (msg: string, ctx?: Record<string, unknown>) => emit('error', msg, ctx),
};
