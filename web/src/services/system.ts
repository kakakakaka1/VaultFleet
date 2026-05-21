import { apiPut, ApiError } from "./http";

export const changePassword = (body: { current_password: string; new_password: string }) => apiPut<{ ok: true }>("/api/system/password", body);

export const exportSystemData = async () => {
  const response = await fetch("/api/system/export", { credentials: "same-origin" });
  if (!response.ok) throw new ApiError("export failed", response.status, await response.text());
  return response.blob();
};

export interface ImportValidationResult {
  valid: boolean;
  files: string[];
  errors: string[];
}

export const importSystemData = async (file: File): Promise<ImportValidationResult> => {
  const formData = new FormData();
  formData.append("file", file);
  const response = await fetch("/api/system/import", {
    method: "POST",
    credentials: "same-origin",
    body: formData,
  });
  if (!response.ok) throw new ApiError("upload failed", response.status, await response.text());
  const body = await response.json();
  if (body.ok === false) throw new ApiError(body.error || "upload failed", response.status, body.error);
  return body.data;
};

export const confirmImport = async (): Promise<void> => {
  const response = await fetch("/api/system/import/confirm", {
    method: "POST",
    credentials: "same-origin",
  });
  if (!response.ok) throw new ApiError("confirm failed", response.status, await response.text());
};
