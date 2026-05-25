export interface ApiResponse<T> {
  ok: boolean;
  data?: T;
  error?: string;
}

export interface AuthUser {
  username: string;
}

export interface AuthCheck {
  authenticated: boolean;
  initialized: boolean;
  username?: string;
  user?: AuthUser;
}

export interface AuthCredentials {
  username: string;
  password?: string;
}

export interface BrowseRequest {
  path: string;
  depth?: number;
}

export interface BrowseEntry {
  path: string;
  type: "file" | "dir";
  size: number;
}

export interface BrowseResponse {
  path: string;
  entries: BrowseEntry[];
}

export interface DirSizeRequest {
  path: string;
}

export interface DirSizeResponse {
  path: string;
  size: number;
  error?: string;
}
