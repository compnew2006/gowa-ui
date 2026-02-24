import { appendFileSync } from 'node:fs';
import type { LogLevel } from './config.js';

const levelWeight: Record<LogLevel, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40
};

export interface LoggerOptions {
  level: LogLevel;
  filePath?: string;
  useStderr?: boolean;
}

export class Logger {
  private readonly level: LogLevel;
  private readonly filePath?: string;
  private readonly useStderr: boolean;

  constructor(options: LoggerOptions) {
    this.level = options.level;
    this.filePath = options.filePath;
    this.useStderr = options.useStderr ?? true;
  }

  debug(message: string, meta?: Record<string, unknown>): void {
    this.write('debug', message, meta);
  }

  info(message: string, meta?: Record<string, unknown>): void {
    this.write('info', message, meta);
  }

  warn(message: string, meta?: Record<string, unknown>): void {
    this.write('warn', message, meta);
  }

  error(message: string, meta?: Record<string, unknown>): void {
    this.write('error', message, meta);
  }

  private write(level: LogLevel, message: string, meta?: Record<string, unknown>): void {
    if (levelWeight[level] < levelWeight[this.level]) {
      return;
    }

    const payload = {
      ts: new Date().toISOString(),
      level,
      message,
      ...(meta ?? {})
    };

    const line = `${JSON.stringify(payload)}\n`;

    if (this.filePath) {
      appendFileSync(this.filePath, line, 'utf8');
      return;
    }

    if (this.useStderr) {
      process.stderr.write(line);
    }
  }
}
