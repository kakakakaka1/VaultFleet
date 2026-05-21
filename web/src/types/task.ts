export interface TaskHistory {
  id: number;
  message_id: string;
  agent_id: string;
  type: "backup" | "restore";
  status: "pending" | "running" | "success" | "failed" | "timeout";
  snapshot_id?: string;
  command_id?: string;
  policy_id?: string;
  storage_id?: string;
  started_at?: string;
  finished_at?: string;
  repo_size?: number;
  duration_ms?: number;
  error_log?: string;
  created_at: string;
  updated_at?: string;
}

export interface TaskFilters {
  agent_id?: string;
  type?: string;
  status?: string;
  limit?: number;
}
