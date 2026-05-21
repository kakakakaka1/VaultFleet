const WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"];

export function describeCron(expr: string): string {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) return expr;

  const [minute, hour, dom, month, dow] = parts;

  try {
    const timePart = describeTime(minute, hour);
    const datePart = describeDate(dom, month, dow);
    return datePart ? `${datePart} ${timePart}` : timePart;
  } catch {
    return expr;
  }
}

function describeTime(minute: string, hour: string): string {
  if (minute.includes("/") && hour === "*") {
    const interval = minute.split("/")[1];
    return `每 ${interval} 分钟`;
  }
  if (hour.includes("/")) {
    const interval = hour.split("/")[1];
    return `每 ${interval} 小时`;
  }
  if (minute === "*" && hour === "*") return "每分钟";

  const h = hour === "*" ? "每小时" : `${hour.padStart(2, "0")}`;
  const m = minute === "*" ? "" : `:${minute.padStart(2, "0")}`;

  if (hour === "*") return `每小时的第 ${minute} 分钟`;
  return `${h}${m}`;
}

function describeDate(dom: string, month: string, dow: string): string {
  const parts: string[] = [];

  if (month !== "*") {
    parts.push(`${month} 月`);
  }

  if (dow !== "*" && dow !== "?") {
    const days = dow.split(",").map((d) => {
      const n = parseInt(d, 10);
      return isNaN(n) ? d : `周${WEEKDAYS[n % 7]}`;
    });
    parts.push(days.join("、"));
  } else if (dom !== "*" && dom !== "?") {
    parts.push(`每月 ${dom} 日`);
  } else if (month === "*") {
    return "每天";
  }

  return parts.join(" ");
}
