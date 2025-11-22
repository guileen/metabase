// Environment variables configuration
export const config = {
	// API configuration
	apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:7610',
	adminBaseUrl: import.meta.env.VITE_ADMIN_BASE_URL || 'http://localhost:7680',

	// Development mode
	isDev: import.meta.env.DEV,

	// Build mode
	mode: import.meta.env.MODE,
} as const;

// Export typed configuration
export type Config = typeof config;