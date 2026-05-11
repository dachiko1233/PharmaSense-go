export interface User {
  id: string;
  email: string;
  full_name: string;
  phone_number?: string;
  sms_enabled: boolean;
  email_verified: boolean;
  default_pharmacy_id: string;
  created_at: string;
}

export interface Pharmacy {
  id: string;
  chain_id?: string;
  name: string;
  license_number: string;
  city?: string;
  phone?: string;
  email?: string;
  plan: "free" | "pro" | "chain";
  language: string;
  subscription_status?: string;
  subscription_current_period_end?: string;
}

export interface InventoryBatch {
  id: string;
  pharmacy_id: string;
  product_id: string;
  batch_number?: string;
  expiry_date: string;
  initial_quantity: number;
  current_quantity: number;
  purchase_price: number;
  selling_price: number;
  supplier?: string;
  received_date: string;
  product_name: string;
  category?: string;
  manufacturer?: string;
  risk_level?: RiskLevel;
  days_until_expiry?: number;
  estimated_loss?: number;
  suggested_discount_percent?: number;
}

export type RiskLevel = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

export interface RiskAssessment {
  id: string;
  batch_id: string;
  pharmacy_id: string;
  risk_level: RiskLevel;
  days_until_expiry: number;
  avg_daily_sales?: number;
  expected_sales?: number;
  estimated_surplus?: number;
  estimated_loss?: number;
  suggested_discount_percent?: number;
  calculated_at: string;
  product_name: string;
  batch_number?: string;
  expiry_date: string;
  current_quantity: number;
  purchase_price: number;
}

export interface DashboardStats {
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
  estimated_loss: number;
  potential_savings: number;
  total_inventory_value: number;
  total_batches: number;
}

export interface TimelinePoint {
  month: string;
  batch_count: number;
  value: number;
}

export interface ChainPharmacyStats {
  id: string;
  name: string;
  plan: string;
  city?: string;
  total_batches: number;
  critical_count: number;
  high_count: number;
  estimated_loss: number;
  inventory_value: number;
}

export interface AlertAction {
  action_type: string;
  discount_percent?: number;
  notes?: string;
}

export interface NotificationSettings {
  email: string;
  full_name: string;
  phone_number?: string;
  sms_enabled: boolean;
  email_verified: boolean;
}

export interface SavingsPoint {
  month: string;
  actions_taken: number;
  savings: number;
}

export interface WastePoint {
  month: string;
  expired_batches: number;
  waste_value: number;
}

export interface CategoryStat {
  category: string;
  batch_count: number;
  at_risk_count: number;
  total_loss: number;
}
