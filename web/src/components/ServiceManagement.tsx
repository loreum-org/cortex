import { useState, useEffect } from 'react';
import { Plus, Trash2, Eye, Server } from 'lucide-react';
import { cortexAPI } from '../services/api';
import type { ServiceOffering, ServiceType, ServiceStatus, RegisterServiceRequest, ServiceCapability } from '../types/services';

export function ServiceManagement() {
  const [services, setServices] = useState<ServiceOffering[]>([]);
  const [selectedService, setSelectedService] = useState<ServiceOffering | null>(null);
  const [showRegisterForm, setShowRegisterForm] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadServices();
  }, []);

  const loadServices = async () => {
    try {
      setLoading(true);
      const nodeServices = await cortexAPI.getNodeServices();
      setServices(nodeServices);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load services');
    } finally {
      setLoading(false);
    }
  };

  const handleDeregister = async (serviceId: string) => {
    if (!confirm('Are you sure you want to deregister this service?')) return;
    
    try {
      await cortexAPI.deregisterService(serviceId);
      await loadServices();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to deregister service');
    }
  };

  const getStatusColor = (status: ServiceStatus) => {
    switch (status) {
      case 'active': return 'text-green-400 bg-green-400/10';
      case 'inactive': return 'text-gray-400 bg-gray-400/10';
      case 'maintenance': return 'text-yellow-400 bg-yellow-400/10';
      case 'error': return 'text-red-400 bg-red-400/10';
      default: return 'text-gray-400 bg-gray-400/10';
    }
  };

  const getTypeColor = (type: ServiceType) => {
    switch (type) {
      case 'agent': return 'text-blue-400 bg-blue-400/10';
      case 'model': return 'text-purple-400 bg-purple-400/10';
      case 'sensor': return 'text-green-400 bg-green-400/10';
      case 'tool': return 'text-orange-400 bg-orange-400/10';
      default: return 'text-gray-400 bg-gray-400/10';
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-tesla-black text-tesla-white p-6">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-tesla-white"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-tesla-black text-tesla-white p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <Server className="h-8 w-8 text-tesla-white" />
            <h1 className="text-3xl font-bold">Service Management</h1>
          </div>
          <button
            onClick={() => setShowRegisterForm(true)}
            className="flex items-center gap-2 bg-tesla-white text-tesla-black px-4 py-2 font-medium hover:bg-tesla-text-gray transition-colors duration-150"
          >
            <Plus className="h-4 w-4" />
            Register Service
          </button>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 mb-6">
            {error}
          </div>
        )}

        {/* Services Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {services.map((service) => (
            <div
              key={service.id}
              className="bg-tesla-dark-gray border border-tesla-border p-6 hover:border-tesla-text-gray transition-colors duration-150"
            >
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold mb-1">{service.name}</h3>
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider ${getTypeColor(service.type)}`}>
                      {service.type}
                    </span>
                    <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider ${getStatusColor(service.status)}`}>
                      {service.status}
                    </span>
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setSelectedService(service)}
                    className="p-2 text-tesla-text-gray hover:text-tesla-white hover:bg-tesla-medium-gray transition-colors duration-150"
                    title="View Details"
                  >
                    <Eye className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDeregister(service.id)}
                    className="p-2 text-tesla-text-gray hover:text-red-400 hover:bg-red-400/10 transition-colors duration-150"
                    title="Deregister"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <p className="text-tesla-text-gray text-sm mb-4 line-clamp-3">
                {service.description}
              </p>

              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-tesla-text-gray">Capabilities:</span>
                  <span className="text-tesla-white">{service.capabilities.length}</span>
                </div>
                
                {service.pricing && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-tesla-text-gray">Base Price:</span>
                    <span className="text-tesla-white">{service.pricing.base_price}</span>
                  </div>
                )}

                <div className="flex items-center justify-between text-sm">
                  <span className="text-tesla-text-gray">Last Seen:</span>
                  <span className="text-tesla-white">
                    {new Date(service.last_seen).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>

        {services.length === 0 && !loading && (
          <div className="text-center py-16">
            <Server className="h-16 w-16 text-tesla-text-gray mx-auto mb-4" />
            <h3 className="text-xl font-medium text-tesla-text-gray mb-2">No Services Registered</h3>
            <p className="text-tesla-text-gray mb-6">Get started by registering your first service.</p>
            <button
              onClick={() => setShowRegisterForm(true)}
              className="bg-tesla-white text-tesla-black px-6 py-3 font-medium hover:bg-tesla-text-gray transition-colors duration-150"
            >
              Register Service
            </button>
          </div>
        )}
      </div>

      {/* Service Details Modal */}
      {selectedService && (
        <ServiceDetailsModal 
          service={selectedService} 
          onClose={() => setSelectedService(null)} 
        />
      )}

      {/* Register Service Modal */}
      {showRegisterForm && (
        <RegisterServiceModal 
          onClose={() => setShowRegisterForm(false)}
          onSuccess={() => {
            setShowRegisterForm(false);
            loadServices();
          }}
        />
      )}
    </div>
  );
}

interface ServiceDetailsModalProps {
  service: ServiceOffering;
  onClose: () => void;
}

function ServiceDetailsModal({ service, onClose }: ServiceDetailsModalProps) {
  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50">
      <div className="bg-tesla-dark-gray border border-tesla-border max-w-2xl w-full max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold">{service.name}</h2>
            <button
              onClick={onClose}
              className="text-tesla-text-gray hover:text-tesla-white transition-colors duration-150"
            >
              ✕
            </button>
          </div>

          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">Description</h3>
              <p className="text-tesla-text-gray">{service.description}</p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <h4 className="font-medium mb-1">Type</h4>
                <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider ${getTypeColor(service.type)}`}>
                  {service.type}
                </span>
              </div>
              <div>
                <h4 className="font-medium mb-1">Status</h4>
                <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider ${getStatusColor(service.status)}`}>
                  {service.status}
                </span>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold mb-3">Capabilities</h3>
              <div className="space-y-3">
                {service.capabilities.map((capability, index) => (
                  <div key={index} className="bg-tesla-medium-gray p-3">
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium">{capability.name}</h4>
                      <span className="text-xs text-tesla-text-gray">v{capability.version}</span>
                    </div>
                    <p className="text-sm text-tesla-text-gray mb-2">{capability.description}</p>
                    {Object.keys(capability.parameters).length > 0 && (
                      <div className="text-xs">
                        <span className="text-tesla-text-gray">Parameters: </span>
                        <span className="text-tesla-white">
                          {Object.keys(capability.parameters).join(', ')}
                        </span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {service.pricing && (
              <div>
                <h3 className="text-lg font-semibold mb-3">Pricing</h3>
                <div className="bg-tesla-medium-gray p-4 space-y-2">
                  <div className="flex justify-between">
                    <span>Base Price:</span>
                    <span>{service.pricing.base_price}</span>
                  </div>
                  {service.pricing.price_per_token && (
                    <div className="flex justify-between">
                      <span>Per Token:</span>
                      <span>{service.pricing.price_per_token}</span>
                    </div>
                  )}
                  {service.pricing.price_per_call && (
                    <div className="flex justify-between">
                      <span>Per Call:</span>
                      <span>{service.pricing.price_per_call}</span>
                    </div>
                  )}
                  {service.pricing.price_per_minute && (
                    <div className="flex justify-between">
                      <span>Per Minute:</span>
                      <span>{service.pricing.price_per_minute}</span>
                    </div>
                  )}
                </div>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-tesla-text-gray">Created:</span>
                <div>{new Date(service.created_at).toLocaleString()}</div>
              </div>
              <div>
                <span className="text-tesla-text-gray">Last Updated:</span>
                <div>{new Date(service.updated_at).toLocaleString()}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface RegisterServiceModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

function RegisterServiceModal({ onClose, onSuccess }: RegisterServiceModalProps) {
  const [formData, setFormData] = useState<RegisterServiceRequest>({
    type: 'agent',
    name: '',
    description: '',
    capabilities: [],
    pricing: {
      base_price: '0.001'
    },
    metadata: {}
  });
  const [newCapability, setNewCapability] = useState<ServiceCapability>({
    name: '',
    description: '',
    parameters: {},
    version: '1.0.0'
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name || !formData.description) {
      setError('Name and description are required');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await cortexAPI.registerService(formData);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to register service');
    } finally {
      setLoading(false);
    }
  };

  const addCapability = () => {
    if (!newCapability.name || !newCapability.description) return;
    setFormData(prev => ({
      ...prev,
      capabilities: [...prev.capabilities, newCapability]
    }));
    setNewCapability({
      name: '',
      description: '',
      parameters: {},
      version: '1.0.0'
    });
  };

  const removeCapability = (index: number) => {
    setFormData(prev => ({
      ...prev,
      capabilities: prev.capabilities.filter((_, i) => i !== index)
    }));
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50">
      <div className="bg-tesla-dark-gray border border-tesla-border max-w-2xl w-full max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold">Register New Service</h2>
            <button
              onClick={onClose}
              className="text-tesla-text-gray hover:text-tesla-white transition-colors duration-150"
            >
              ✕
            </button>
          </div>

          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 mb-6">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">Service Type</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData(prev => ({ ...prev, type: e.target.value as ServiceType }))}
                  className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                >
                  <option value="agent">Agent</option>
                  <option value="model">Model</option>
                  <option value="sensor">Sensor</option>
                  <option value="tool">Tool</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Base Price</label>
                <input
                  type="text"
                  value={formData.pricing?.base_price || ''}
                  onChange={(e) => setFormData(prev => ({
                    ...prev,
                    pricing: { ...prev.pricing, base_price: e.target.value }
                  }))}
                  className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                  placeholder="0.001"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Service Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                placeholder="Enter service name"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                rows={3}
                placeholder="Describe your service"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-3">Capabilities</label>
              
              {/* Add Capability Form */}
              <div className="bg-tesla-medium-gray p-4 mb-4 space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <input
                    type="text"
                    value={newCapability.name}
                    onChange={(e) => setNewCapability(prev => ({ ...prev, name: e.target.value }))}
                    className="bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                    placeholder="Capability name"
                  />
                  <input
                    type="text"
                    value={newCapability.version}
                    onChange={(e) => setNewCapability(prev => ({ ...prev, version: e.target.value }))}
                    className="bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                    placeholder="Version"
                  />
                </div>
                <textarea
                  value={newCapability.description}
                  onChange={(e) => setNewCapability(prev => ({ ...prev, description: e.target.value }))}
                  className="w-full bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-text-gray"
                  rows={2}
                  placeholder="Capability description"
                />
                <button
                  type="button"
                  onClick={addCapability}
                  className="bg-tesla-white text-tesla-black px-4 py-2 text-sm font-medium hover:bg-tesla-text-gray transition-colors duration-150"
                >
                  Add Capability
                </button>
              </div>

              {/* Capabilities List */}
              <div className="space-y-2">
                {formData.capabilities.map((capability, index) => (
                  <div key={index} className="bg-tesla-black border border-tesla-border p-3 flex items-center justify-between">
                    <div>
                      <div className="font-medium">{capability.name} v{capability.version}</div>
                      <div className="text-sm text-tesla-text-gray">{capability.description}</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeCapability(index)}
                      className="text-red-400 hover:text-red-300 transition-colors duration-150"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex gap-4 pt-4">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 bg-tesla-medium-gray text-tesla-white px-4 py-3 font-medium hover:bg-tesla-text-gray transition-colors duration-150"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 bg-tesla-white text-tesla-black px-4 py-3 font-medium hover:bg-tesla-text-gray transition-colors duration-150 disabled:opacity-50"
              >
                {loading ? 'Registering...' : 'Register Service'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

function getTypeColor(type: ServiceType) {
  switch (type) {
    case 'agent': return 'text-blue-400 bg-blue-400/10';
    case 'model': return 'text-purple-400 bg-purple-400/10';
    case 'sensor': return 'text-green-400 bg-green-400/10';
    case 'tool': return 'text-orange-400 bg-orange-400/10';
    default: return 'text-gray-400 bg-gray-400/10';
  }
}

function getStatusColor(status: ServiceStatus) {
  switch (status) {
    case 'active': return 'text-green-400 bg-green-400/10';
    case 'inactive': return 'text-gray-400 bg-gray-400/10';
    case 'maintenance': return 'text-yellow-400 bg-yellow-400/10';
    case 'error': return 'text-red-400 bg-red-400/10';
    default: return 'text-gray-400 bg-gray-400/10';
  }
}