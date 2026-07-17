// Mirrors backend/internal/modules/installation/model.go — add one type per
// domain module as features are built.
export interface Installation {
  id: number;
  tenant_id: number;
  installed: boolean;
  created_at: string;
  updated_at: string;
}
