import { hashIndex } from "@/utils/color-hash";

export interface ServiceVisual {
  bg: string;
  fg: string;
  solid: string;
}

// Service names are open-ended (admin-defined), so chip/timeline colors are
// assigned by hashing the name into a fixed palette rather than matching on
// specific service names — every service gets a stable color, including
// ones that don't exist yet.
const SERVICE_VISUAL_PALETTE: ServiceVisual[] = [
  { bg: "#F5F3FF", fg: "#7C3AED", solid: "#8B5CF6" },
  { bg: "#EFF6FF", fg: "#2563EB", solid: "#3B82F6" },
  { bg: "#ECFDF5", fg: "#059669", solid: "#10B981" },
  { bg: "#FFF7ED", fg: "#C2410C", solid: "#F97316" },
  { bg: "#FDF2F8", fg: "#DB2777", solid: "#EC4899" },
  { bg: "#ECFEFF", fg: "#0891B2", solid: "#06B6D4" },
];

export function serviceVisual(serviceName: string): ServiceVisual {
  return SERVICE_VISUAL_PALETTE[hashIndex(serviceName, SERVICE_VISUAL_PALETTE.length)];
}
