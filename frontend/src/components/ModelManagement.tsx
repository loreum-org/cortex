import { useState, useEffect } from 'react';
import { Download, RefreshCw, Brain, Trash2 } from 'lucide-react';
import { Terminal } from './Terminal';
import { cortexWebSocket } from '../services/websocket';

interface OllamaModel {
  name: string;
  size: string;
  digest: string;
  modified: string;
}

interface AvailableModel {
  name: string;
  description: string;
  size: string;
  tags: string[];
  downloads: number;
}

export function ModelManagement() {
  const [installedModels, setInstalledModels] = useState<OllamaModel[]>([]);
  const [availableModels, setAvailableModels] = useState<AvailableModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [downloading, setDownloading] = useState<Set<string>>(new Set());
  const [deleting, setDeleting] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadModels();
  }, []);

  const loadModels = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Get models via WebSocket API
      const modelsData = await cortexWebSocket.getModels();
      
      console.log('📊 Raw models response:', modelsData);
      
      // Set installed models
      if (modelsData.installed && Array.isArray(modelsData.installed)) {
        setInstalledModels(modelsData.installed);
      } else {
        console.warn('⚠️ No installed models data in response:', modelsData.installed);
        setInstalledModels([]);
      }
      
      // Set available models
      if (modelsData.available && Array.isArray(modelsData.available)) {
        setAvailableModels(modelsData.available);
      } else {
        console.warn('⚠️ No available models data in response:', modelsData.available);
        setAvailableModels([]);
      }
    } catch (err) {
      console.error('Failed to load models:', err);
      setError(err instanceof Error ? err.message : 'Failed to load models');
    } finally {
      setLoading(false);
    }
  };

  const downloadModel = async (modelName: string) => {
    try {
      setDownloading(prev => new Set(prev).add(modelName));
      setError(null);
      
      // Download via WebSocket API with progress tracking
      await cortexWebSocket.downloadModel(modelName, (progress, message) => {
        console.log(`Download progress for ${modelName}: ${progress}% - ${message}`);
      });

      // Reload models after download
      loadModels();
      
      setDownloading(prev => {
        const newSet = new Set(prev);
        newSet.delete(modelName);
        return newSet;
      });

    } catch (err) {
      setError(`Failed to download ${modelName}: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setDownloading(prev => {
        const newSet = new Set(prev);
        newSet.delete(modelName);
        return newSet;
      });
    }
  };

  const deleteModel = async (modelName: string) => {
    if (!confirm(`Are you sure you want to delete the model "${modelName}"?`)) return;
    
    try {
      setDeleting(prev => new Set(prev).add(modelName));
      setError(null);
      
      // Delete model via WebSocket API
      await cortexWebSocket.deleteModel(modelName);

      // Remove from installed models
      setInstalledModels(prev => prev.filter(model => model.name !== modelName));
    } catch (err) {
      setError(`Failed to delete ${modelName}: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setDeleting(prev => {
        const newSet = new Set(prev);
        newSet.delete(modelName);
        return newSet;
      });
    }
  };

  return (
    <div className="min-h-screen bg-tesla-black text-tesla-white p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <Brain className="h-8 w-8 text-tesla-white" />
            <h1 className="text-3xl font-bold">Model Management</h1>
          </div>
          <button
            onClick={loadModels}
            className="flex items-center gap-2 px-4 py-2 bg-tesla-medium-gray border border-tesla-border hover:border-tesla-text-gray transition-colors"
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-6 mb-8">
          {/* Models Content - Takes 2/3 of the space */}
          <div className="xl:col-span-2">
            {/* Error Display */}
            {error && (
              <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 mb-6">
                {error}
              </div>
            )}

            <div className="space-y-8">
              {/* Installed Models */}
              <div className="bg-tesla-dark-gray border border-tesla-border p-6">
                <h2 className="text-xl font-semibold mb-4">Installed Models ({installedModels.length})</h2>

                {loading ? (
                  <div className="flex justify-center py-8">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-tesla-white"></div>
                  </div>
                ) : installedModels.length === 0 ? (
                  <div className="text-center py-8 text-tesla-text-gray">
                    <Brain className="h-16 w-16 mx-auto mb-4 opacity-50" />
                    <p>No models installed. Download models from the available section below.</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {installedModels.map((model) => (
                      <div key={model.name} className="bg-tesla-medium-gray border border-tesla-border p-4">
                        <div className="flex items-center justify-between mb-2">
                          <h3 className="font-semibold text-tesla-white">{model.name}</h3>
                          <button
                            onClick={() => deleteModel(model.name)}
                            className="text-red-400 hover:text-red-300 transition-colors"
                            disabled={deleting.has(model.name)}
                            title="Delete model"
                          >
                            {deleting.has(model.name) ? (
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            ) : (
                              <Trash2 className="h-4 w-4" />
                            )}
                          </button>
                        </div>
                        <div className="text-sm text-tesla-text-gray space-y-1">
                          <div>Size: {model.size}</div>
                          <div>Modified: {new Date(model.modified).toLocaleDateString()}</div>
                          <div className="text-xs text-green-400">● Available</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Available Models */}
              <div className="bg-tesla-dark-gray border border-tesla-border p-6">
                <h2 className="text-xl font-semibold mb-4">Available Models</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {availableModels
                    .filter((model) => {
                      // Hide models that are already installed
                      const isInstalled = installedModels.some(installed => {
                        // Direct match
                        if (installed.name === model.name) return true;
                        // Match without :latest suffix
                        if (installed.name === model.name + ':latest') return true;
                        if (installed.name.replace(':latest', '') === model.name) return true;
                        // Match base name (before first colon)
                        const installedBase = installed.name.split(':')[0];
                        const availableBase = model.name.split(':')[0];
                        return installedBase === availableBase && installed.name.includes(model.name.split(':')[1] || '');
                      });
                      return !isInstalled; // Only show models that are NOT installed
                    })
                    .map((model) => {
                    const isDownloading = downloading.has(model.name);
                    
                    return (
                      <div key={model.name} className="bg-tesla-medium-gray border border-tesla-border p-4">
                        <div className="mb-3">
                          <h3 className="font-semibold text-tesla-white mb-1">{model.name}</h3>
                          <p className="text-sm text-tesla-text-gray mb-2">{model.description}</p>
                          <div className="flex flex-wrap gap-1 mb-2">
                            {model.tags.map(tag => (
                              <span key={tag} className="px-2 py-1 bg-tesla-black text-xs rounded">
                                {tag}
                              </span>
                            ))}
                          </div>
                          <div className="text-sm text-tesla-text-gray space-y-1">
                            <div>Size: {model.size}</div>
                            <div>Downloads: {model.downloads.toLocaleString()}</div>
                          </div>
                        </div>
                        
                        <button
                          onClick={() => downloadModel(model.name)}
                          disabled={isDownloading}
                          className={`w-full flex items-center justify-center gap-2 px-4 py-2 transition-colors ${
                            isDownloading
                              ? 'bg-blue-600/20 text-blue-400 cursor-not-allowed'
                              : 'bg-tesla-white text-tesla-black hover:bg-tesla-text-gray'
                          }`}
                        >
                          {isDownloading ? (
                            <>
                              <RefreshCw className="h-4 w-4 animate-spin" />
                              Downloading...
                            </>
                          ) : (
                            <>
                              <Download className="h-4 w-4" />
                              Download
                            </>
                          )}
                        </button>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>

          {/* Terminal - Takes 1/3 of the space */}
          <div className="xl:col-span-1">
            <Terminal 
              title="Ollama Logs" 
              height="h-[600px]" 
              className="sticky top-6"
            />
          </div>
        </div>
      </div>
    </div>
  );
}