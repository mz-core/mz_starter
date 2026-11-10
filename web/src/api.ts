export const token = location.hash.slice(1);

export async function api<T>(path: string, body?: object): Promise<T> {
  const response = await fetch(path, {
    method: body ? 'POST' : 'GET',
    headers: { 'X-MZ-Session': token, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ code: 'network_error' }));
    throw new Error(data.code || 'network_error');
  }
  return response.json() as Promise<T>;
}
