// Types for the '@root/package.json' alias (see vite.config.ts) which points
// at the repo-root package.json - the single source of truth for the version.
declare module '@root/package.json' {
	const value: {
		name: string;
		version: string;
		[key: string]: unknown;
	};
	export default value;
}
