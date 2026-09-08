export const RISK_COLORS = {
  LOW: '#16A34A',
  MODERATE: '#FACC15',
  HIGH: '#F97316',
  VERY_HIGH: '#DC2626',
  EXTREME: '#7C1D6F',
} as const;

export const RISK_LABEL = {
  LOW: 'Low',
  MODERATE: 'Moderate',
  HIGH: 'High',
  VERY_HIGH: 'Very High',
  EXTREME: 'Extreme',
} as const;

export function riskColor(band: string): string {
  return (RISK_COLORS as any)[band] ?? '#94A3B8';
}