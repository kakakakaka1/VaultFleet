export interface HealthResponse {
  ok: boolean;
  status: string;
}

export interface ReadyResponse {
  ok: boolean;
  status?: string;
  error?: string;
}

export interface StorageTestResult {
  ok: boolean;
  latency_ms: number;
  error?: string;
  checked_at: string;
}
