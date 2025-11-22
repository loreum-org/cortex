import { useState, useEffect, useRef } from 'react';
import { Terminal as TerminalIcon, X, Maximize2, Minimize2, Copy } from 'lucide-react';
import { cortexWebSocket } from '../services/websocket';

interface TerminalProps {
  title?: string;
  height?: string;
  className?: string;
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  source: string;
}

export function Terminal({ title = "System Logs", height = "h-96", className = "" }: TerminalProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isMinimized, setIsMinimized] = useState(false);
  const [isMaximized, setIsMaximized] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const terminalRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Subscribe to log events using the shared WebSocket service
    const unsubscribeOllamaLogs = cortexWebSocket.subscribe('ollama_logs', (data) => {
      const logEntry: LogEntry = {
        timestamp: new Date().toISOString(),
        level: data?.level || 'INFO',
        message: data?.message || data || 'Unknown ollama log entry',
        source: 'ollama'
      };
      
      setLogs(prev => {
        const newLogs = [...prev, logEntry];
        // Keep only last 1000 logs to prevent memory issues
        return newLogs.slice(-1000);
      });
    });

    const unsubscribeCortexLogs = cortexWebSocket.subscribe('cortex_logs', (data) => {
      const logEntry: LogEntry = {
        timestamp: new Date().toISOString(),
        level: data?.level || 'INFO',
        message: data?.message || data || 'Unknown cortex log entry',
        source: 'cortex'
      };
      
      setLogs(prev => {
        const newLogs = [...prev, logEntry];
        return newLogs.slice(-1000);
      });
    });

    const unsubscribeLogEntry = cortexWebSocket.subscribe('log_entry', (data) => {
      const logEntry: LogEntry = {
        timestamp: new Date().toISOString(),
        level: data?.level || 'INFO',
        message: data?.message || data || 'Unknown log entry',
        source: data?.source || 'system'
      };
      
      setLogs(prev => {
        const newLogs = [...prev, logEntry];
        return newLogs.slice(-1000);
      });
    });

    // Subscribe to connection status changes
    const unsubscribeStatus = cortexWebSocket.onConnectionStatusChange((status) => {
      setConnectionStatus(status);
    });

    // Get initial connection status
    setConnectionStatus(cortexWebSocket.getConnectionStatus());

    // Subscribe to log streams via the shared WebSocket
    cortexWebSocket.sendMessage({
      type: 'subscribe',
      data: 'ollama_logs'
    });
    cortexWebSocket.sendMessage({
      type: 'subscribe',
      data: 'cortex_logs'
    });

    // Cleanup function
    return () => {
      unsubscribeOllamaLogs();
      unsubscribeCortexLogs();
      unsubscribeLogEntry();
      unsubscribeStatus();
    };
  }, []);

  useEffect(() => {
    // Auto-scroll to bottom when new logs arrive
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const handleCopyLogs = () => {
    const logText = logs.map(log => 
      `[${new Date(log.timestamp).toLocaleTimeString()}] ${log.level}: ${log.message}`
    ).join('\n');
    
    navigator.clipboard.writeText(logText).then(() => {
      // Could show a toast notification here
      console.log('Logs copied to clipboard');
    });
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const getLevelColor = (level: string) => {
    switch (level.toUpperCase()) {
      case 'ERROR': return 'text-red-400';
      case 'WARN': return 'text-yellow-400';
      case 'INFO': return 'text-blue-400';
      case 'DEBUG': return 'text-gray-400';
      default: return 'text-tesla-white';
    }
  };

  if (isMinimized) {
    return (
      <div className={`bg-tesla-dark-gray border border-tesla-border ${className}`}>
        <div className="flex items-center justify-between p-2 bg-tesla-medium-gray border-b border-tesla-border">
          <div className="flex items-center gap-2">
            <TerminalIcon className="h-4 w-4 text-tesla-white" />
            <span className="text-sm font-medium text-tesla-white">{title}</span>
            <span className="text-xs text-tesla-text-gray">({logs.length} logs)</span>
          </div>
          <button
            onClick={() => setIsMinimized(false)}
            className="text-tesla-text-gray hover:text-tesla-white"
          >
            <Maximize2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    );
  }

  const terminalHeight = isMaximized ? 'h-screen' : height;
  const terminalClass = isMaximized 
    ? 'fixed inset-0 z-50 bg-tesla-black' 
    : `${className} bg-tesla-dark-gray border border-tesla-border`;

  return (
    <div className={terminalClass}>
      {/* Terminal Header */}
      <div className="flex items-center justify-between p-3 bg-tesla-medium-gray border-b border-tesla-border">
        <div className="flex items-center gap-2">
          <TerminalIcon className="h-4 w-4 text-tesla-white" />
          <span className="text-sm font-medium text-tesla-white">{title}</span>
          <span className="text-xs text-tesla-text-gray">({logs.length} logs)</span>
        </div>
        
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-2 text-xs text-tesla-text-gray">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded bg-tesla-black border-tesla-border"
            />
            Auto-scroll
          </label>
          
          <button
            onClick={handleCopyLogs}
            className="text-tesla-text-gray hover:text-tesla-white"
            title="Copy logs"
          >
            <Copy className="h-4 w-4" />
          </button>
          
          <button
            onClick={() => setIsMinimized(true)}
            className="text-tesla-text-gray hover:text-tesla-white"
            title="Minimize"
          >
            <Minimize2 className="h-4 w-4" />
          </button>
          
          <button
            onClick={() => setIsMaximized(!isMaximized)}
            className="text-tesla-text-gray hover:text-tesla-white"
            title={isMaximized ? "Restore" : "Maximize"}
          >
            <Maximize2 className="h-4 w-4" />
          </button>
          
          {isMaximized && (
            <button
              onClick={() => setIsMaximized(false)}
              className="text-tesla-text-gray hover:text-tesla-white"
              title="Close"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Terminal Content */}
      <div 
        ref={terminalRef}
        className={`${terminalHeight} overflow-auto bg-tesla-black p-4 font-mono text-sm`}
        onScroll={(e) => {
          const element = e.target as HTMLDivElement;
          const isAtBottom = element.scrollHeight - element.scrollTop === element.clientHeight;
          setAutoScroll(isAtBottom);
        }}
      >
        {logs.length === 0 ? (
          <div className="text-tesla-text-gray text-center py-8">
            <TerminalIcon className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>Waiting for system logs...</p>
            <p className="text-xs mt-1">Log stream will appear here in real-time</p>
          </div>
        ) : (
          <div className="space-y-1">
            {logs.map((log, index) => (
              <div key={index} className="flex gap-3 text-xs leading-relaxed">
                <span className="text-tesla-text-gray whitespace-nowrap">
                  {formatTimestamp(log.timestamp)}
                </span>
                <span className={`${getLevelColor(log.level)} whitespace-nowrap min-w-[3rem]`}>
                  {log.level.toUpperCase()}
                </span>
                <span className="text-tesla-white flex-1 break-words">
                  {log.message}
                </span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Terminal Footer */}
      <div className="px-4 py-2 bg-tesla-medium-gray border-t border-tesla-border">
        <div className="flex items-center justify-between text-xs text-tesla-text-gray">
          <div className="flex items-center gap-4">
            <span>Lines: {logs.length}</span>
            <span>Auto-scroll: {autoScroll ? 'ON' : 'OFF'}</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              <div className={`w-2 h-2 rounded-full ${
                connectionStatus === 'connected' ? 'bg-green-400' : 
                connectionStatus === 'connecting' ? 'bg-yellow-400' : 
                'bg-red-400'
              }`}></div>
              <span className="capitalize">{connectionStatus}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}