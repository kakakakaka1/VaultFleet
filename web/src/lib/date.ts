import { format } from "date-fns";
import { zhCN } from "date-fns/locale";

export function safeFormatDate(value: string | null | undefined, pattern: string, fallback = "-"): string {
  if (!value) {
    return fallback;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return fallback;
  }
  return format(date, pattern, { locale: zhCN });
}
