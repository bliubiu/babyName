const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function generateNames(data: any): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/names/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return response.json();
}

export async function getHistory(): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/history`);
  return response.json();
}

export async function saveHistory(data: any): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/history`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return response.json();
}

export async function deleteHistory(id: string): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/history/${id}`, {
    method: 'DELETE',
  });
  return response.json();
}

export async function getHexagrams(): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/yijing/hexagram`);
  return response.json();
}

export async function getZodiacs(): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/zodiac`);
  return response.json();
}

export interface FavoriteData {
  id?: string;
  surname: string;
  given_name: string;
  pinyin: string;
  gender: string;
  score: number;
  source?: string;
  notes?: string;
}

export async function getFavorites(): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/favorites`);
  return response.json();
}

export async function saveFavorite(data: FavoriteData): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/favorites`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return response.json();
}

export async function deleteFavorite(id: string): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/favorites/${id}`, {
    method: 'DELETE',
  });
  return response.json();
}

export async function checkFavorite(surname: string, givenName: string): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/favorites/check?surname=${surname}&given_name=${givenName}`);
  return response.json();
}

export interface NameStat {
  name: string;
  count: number;
  rate: number;
  rank: number;
  province?: string;
  year_range?: string;
}

export async function getNameStats(name: string): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/namestat/${encodeURIComponent(name)}`);
  return response.json();
}

export async function getProvinceStats(name: string, province: string = '北京'): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/namestat/${encodeURIComponent(name)}/province?province=${encodeURIComponent(province)}`);
  return response.json();
}

export async function getTopNames(limit: number = 20): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/namestat?limit=${limit}`);
  return response.json();
}
