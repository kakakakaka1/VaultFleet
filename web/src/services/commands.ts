import { AgentCommand, CommandFilters } from "@/types/command";
import { apiGet } from "./http";

export const getCommand = (id: string) => apiGet<AgentCommand>(`/api/commands/${id}`);

export const listAgentCommands = (agentId: string, filters: CommandFilters = {}) =>
  apiGet<AgentCommand[]>(`/api/agents/${agentId}/commands${toQuery(filters)}`);

function toQuery(filters: CommandFilters): string {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.type) params.set("type", filters.type);
  if (filters.limit) params.set("limit", filters.limit.toString());
  const query = params.toString();
  return query ? `?${query}` : "";
}
