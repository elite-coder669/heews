import type { Ward } from '../types';

const FALLBACK_BOX = 0.012;

type Ring = [number, number][];

let cache: Record<string, Ring> | null = null;

async function loadGeoJSON(): Promise<any | null> {
  try {
    const r = await fetch('/data/hyderabad_wards.geojson');
    if (!r.ok) return null;
    return await r.json();
  } catch { return null; }
}

export async function fetchWardPolygons(): Promise<Record<string, Ring>> {
  const parsed = await loadGeoJSON();
  const out: Record<string, Ring> = {};
  if (parsed?.features) {
    for (const f of parsed.features) {
      const wn = Number(f.properties?.ward_number ?? 0);
      const id = 'ward_' + wn.toString().padStart(3, '0');
      const geom = f.geometry;
      if (!geom) continue;
      if (geom.type === 'Polygon') {
        const ring: number[][] = geom.coordinates[0];
        out[id] = ring.map((c) => [c[1], c[0]]) as Ring;
      } else if (geom.type === 'MultiPolygon') {
        const ring: number[][] = geom.coordinates[0][0];
        out[id] = ring.map((c) => [c[1], c[0]]) as Ring;
      }
    }
  }
  cache = out;
  return out;
}

export function fallbackBoxes(wards: Ward[]): Record<string, Ring> {
  const out: Record<string, Ring> = {};
  for (const w of wards) {
    if (cache?.[w.ward_id]) { out[w.ward_id] = cache[w.ward_id]; continue; }
    const { lat, lon } = w.centroid;
    out[w.ward_id] = [
      [lat - FALLBACK_BOX, lon - FALLBACK_BOX],
      [lat + FALLBACK_BOX, lon - FALLBACK_BOX],
      [lat + FALLBACK_BOX, lon + FALLBACK_BOX],
      [lat - FALLBACK_BOX, lon + FALLBACK_BOX],
      [lat - FALLBACK_BOX, lon - FALLBACK_BOX],
    ];
  }
  return out;
}