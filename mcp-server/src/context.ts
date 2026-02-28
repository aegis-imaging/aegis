import { AegisApiClient } from "./aegisClient.js";
import { loadConfig } from "./config.js";

export const config = loadConfig();
export const client = new AegisApiClient(config.aegisApiBaseUrl, config.aegisApiToken);
