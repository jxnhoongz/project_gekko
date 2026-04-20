export function formatDate(iso: string, opts?: Intl.DateTimeFormatOptions): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, opts ?? { year: 'numeric', month: 'short', day: 'numeric' });
}

export function formatDateTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

export function timeAgo(iso: string): string {
  const then = new Date(iso).getTime();
  const now = Date.now();
  const diffSec = Math.round((now - then) / 1000);
  const abs = Math.abs(diffSec);
  const sign = diffSec >= 0 ? '' : 'in ';
  const suffix = diffSec >= 0 ? ' ago' : '';

  if (abs < 60)       return sign + abs + 's' + suffix;
  if (abs < 3600)     return sign + Math.round(abs / 60) + 'm' + suffix;
  if (abs < 86400)    return sign + Math.round(abs / 3600) + 'h' + suffix;
  if (abs < 86400*7)  return sign + Math.round(abs / 86400) + 'd' + suffix;
  if (abs < 86400*30) return sign + Math.round(abs / (86400*7)) + 'w' + suffix;
  return sign + Math.round(abs / (86400*30)) + 'mo' + suffix;
}

export function ageFromBirth(iso: string): string {
  const born = new Date(iso);
  const now = new Date();
  const months =
    (now.getFullYear() - born.getFullYear()) * 12 + (now.getMonth() - born.getMonth());
  if (months < 12) return `${months} mo`;
  const years = Math.floor(months / 12);
  const rem = months % 12;
  return rem === 0 ? `${years} yr` : `${years} yr ${rem} mo`;
}
