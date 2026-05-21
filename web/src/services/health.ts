import { HealthResponse, ReadyResponse } from "@/types/health";

export async function checkHealth(): Promise<HealthResponse> {
  const response = await fetch("/health");
  return response.json();
}

export async function checkReady(): Promise<ReadyResponse> {
  const response = await fetch("/ready");
  return response.json();
}
