export interface ServiceCapability {
  name: string;
  description: string;
  parameters: Record<string, string>;
  version: string;
}

export interface ServicePricing {
  base_price: string;
  price_per_token?: string;
  price_per_call?: string;
  price_per_minute?: string;
}

export type ServiceType = 'agent' | 'model' | 'sensor' | 'tool';
export type ServiceStatus = 'active' | 'inactive' | 'maintenance' | 'error';

export interface ServiceOffering {
  id: string;
  node_id: string;
  type: ServiceType;
  name: string;
  description: string;
  capabilities: ServiceCapability[];
  pricing?: ServicePricing;
  metadata?: Record<string, any>;
  status: ServiceStatus;
  created_at: string;
  updated_at: string;
  last_seen: string;
}

export interface ServiceRequirement {
  type: ServiceType;
  capabilities: string[];
  min_version?: string;
  preferences?: Record<string, any>;
}

export interface ServiceAttestation {
  id: string;
  query_id: string;
  node_id: string;
  service_id: string;
  user_id: string;
  start_time: string;
  end_time: string;
  input_hash: string;
  output_hash: string;
  success: boolean;
  error_msg?: string;
  tokens_used?: number;
  compute_time_ms: number;
  cost: string;
  signature: string;
  created_at: string;
}

export interface RegisterServiceRequest {
  type: ServiceType;
  name: string;
  description: string;
  capabilities: ServiceCapability[];
  pricing?: ServicePricing;
  metadata?: Record<string, any>;
}

export interface NetworkServiceStats {
  services: Record<string, ServiceOffering>;
  by_type: Record<ServiceType, ServiceOffering[]>;
  total: number;
  timestamp: string;
}