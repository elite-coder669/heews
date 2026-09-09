import { useMemo, useEffect } from 'react';
import { MapContainer, TileLayer, Polygon, CircleMarker, Tooltip, useMap } from 'react-leaflet';
import type { LatLngExpression } from 'leaflet';
import type { Ward } from '../types';
import { riskColor } from '../styles/tokens';

interface Props {
  wards: Ward[];
  risks: Record<string, { band: string; score: number; wbgt?: number; utci?: number }>;
  selectedId: string | null;
  onSelect: (id: string) => void;
  polygons?: Record<string, [number, number][]>;
  focusId?: string | null;
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

function FocusController({ focusId, wards }: { focusId?: string | null; wards: Ward[] }) {
  const map = useMap();
  useEffect(() => {
    if (!focusId) return;
    const w = wards.find((x) => x.ward_id === focusId);
    if (!w) return;
    map.setView([w.centroid.lat, w.centroid.lon], Math.max(map.getZoom(), 13), { animate: false });
  }, [focusId, wards, map]);
  return null;
}

export default function RiskMap({ wards, risks, selectedId, onSelect, polygons, focusId }: Props) {
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
      <FocusController focusId={focusId} wards={wards} />
      {wards.map((w) => {
        const ring = polygons?.[w.ward_id];
        if (!ring || ring.length < 3) return null;
        const positions = ring as LatLngExpression[];
        const band = risks[w.ward_id]?.band ?? 'LOW';
        const r = risks[w.ward_id];
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
              <div style={{ fontSize: 12, minWidth: 120 }}>
                <strong>Ward #{w.ward_number}</strong> · {w.ward_name}
                <br />MRI: <strong>{r?.score?.toFixed(0) ?? '—'}</strong>
                <br />WBGT: {r?.wbgt !== undefined ? `${r.wbgt.toFixed(1)} °C` : '—'}
                <br />UTCI: {r?.utci !== undefined ? `${r.utci.toFixed(1)} °C` : '—'}
                <br />Band: <strong>{band}</strong>
              </div>
            </Tooltip>
          </Polygon>
        );
      })}
      {wards.map((w) => (
        <CircleMarker
          key={`c-${w.ward_id}`}
          center={[w.centroid.lat, w.centroid.lon]}
          radius={w.ward_id === selectedId ? 7 : 5}
          pathOptions={{
            color: '#0F172A',
            weight: 1.5,
            fillColor: riskColor(risks[w.ward_id]?.band ?? 'LOW'),
            fillOpacity: 0.9,
          }}
          eventHandlers={{ click: () => onSelect(w.ward_id) }}
        />
      ))}
    </MapContainer>
  );
}