import { useMemo } from 'react';
import { MapContainer, TileLayer, Polygon, Tooltip, useMap } from 'react-leaflet';
import type { LatLngExpression } from 'leaflet';
import type { Ward } from '../types';
import { riskColor } from '../styles/tokens';

interface Props {
  wards: Ward[];
  risks: Record<string, { band: string; score: number }>;
  selectedId: string | null;
  onSelect: (id: string) => void;
  polygons?: Record<string, [number, number][]>;
}

function FitBounds({ wards }: { wards: Ward[] }) {
  const map = useMap();
  useMemo(() => {
    if (!wards.length) return;
    const lats = wards.map((w) => w.centroid.lat);
    const lons = wards.map((w) => w.centroid.lon);
    const south = Math.min(...lats) - 0.05;
    const north = Math.max(...lats) + 0.05;
    const west = Math.min(...lons) - 0.05;
    const east = Math.max(...lons) + 0.05;
    map.fitBounds([[south, west], [north, east]]);
  }, [wards, map]);
  return null;
}

export default function RiskMap({ wards, risks, selectedId, onSelect, polygons }: Props) {
  const center: LatLngExpression = [17.3850, 78.4867];
  const selectedWard = wards.find((w) => w.ward_id === selectedId);
  const focus: LatLngExpression = selectedWard
    ? [selectedWard.centroid.lat, selectedWard.centroid.lon]
    : center;

  return (
    <MapContainer center={focus} zoom={11} style={{ height: '100%', width: '100%' }}>
      <TileLayer
        attribution='&copy; OpenStreetMap contributors'
        url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      <FitBounds wards={wards} />
      {wards.map((w) => {
        const ring = polygons?.[w.ward_id];
        if (!ring || ring.length < 3) return null;
        const positions = ring as LatLngExpression[];
        const band = risks[w.ward_id]?.band ?? 'LOW';
        const fill = riskColor(band);
        const isSelected = w.ward_id === selectedId;
        return (
          <Polygon
            key={w.ward_id}
            positions={positions}
            pathOptions={{
              color: isSelected ? '#0F172A' : '#FFFFFF',
              weight: isSelected ? 3 : 1,
              fillColor: fill,
              fillOpacity: isSelected ? 0.85 : 0.65,
            }}
            eventHandlers={{ click: () => onSelect(w.ward_id) }}
          >
            <Tooltip direction="top" sticky>
              <div style={{ fontSize: 12 }}>
                <strong>{w.ward_name}</strong> · {w.zone}
                <br />Band: <strong>{band}</strong>
                <br />MRI: {risks[w.ward_id]?.score?.toFixed(0) ?? '—'}
              </div>
            </Tooltip>
          </Polygon>
        );
      })}
    </MapContainer>
  );
}