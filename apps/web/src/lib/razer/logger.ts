export type LogLevel = 'info' | 'send' | 'recv' | 'warn' | 'error';

export interface LogEntry {
	id: number;
	ts: number;
	level: LogLevel;
	message: string;
	hex?: Uint8Array;
	direction?: 'tx' | 'rx';
}

export class Logger {
	private entries: LogEntry[] = [];
	private max = 400;
	private counter = 0;

	log = (level: LogLevel, message: string, hex?: Uint8Array, direction?: 'tx' | 'rx') => {
		const entry: LogEntry = { id: ++this.counter, ts: performance.now(), level, message, hex, direction };
		this.entries.push(entry);
		if (this.entries.length > this.max) {
			this.entries.splice(0, this.entries.length - this.max);
		}
	};

	info(message: string) {
		this.log('info', message);
	}
	warn(message: string) {
		this.log('warn', message);
	}
	error(message: string) {
		this.log('error', message);
	}
	tx(message: string, hex?: Uint8Array) {
		this.log('send', message, hex, 'tx');
	}
	rx(message: string, hex?: Uint8Array) {
		this.log('recv', message, hex, 'rx');
	}

	get all(): LogEntry[] {
		return [...this.entries];
	}

	clear() {
		this.entries = [];
	}
}

export const logger = new Logger();
