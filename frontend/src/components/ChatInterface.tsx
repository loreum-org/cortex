import { useState, useRef, useEffect } from 'react';
import { Send, Bot, User, Loader2, Database, AlertCircle, Wifi, WifiOff } from 'lucide-react';
import { marked } from 'marked';
import { cortexWebSocket, type ConnectionStatus } from '../services/websocket';
import type { ChatMessage } from '../services/api';

export function ChatInterface() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [useRAG, setUseRAG] = useState(true);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected');
  const [isStreaming, setIsStreaming] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [hasUserInteracted, setHasUserInteracted] = useState(false);
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [historyLoaded, setHistoryLoaded] = useState(false);
  
  // Enhanced state for consciousness and metadata
  const [consciousnessState, setConsciousnessState] = useState<any>(null);
  const [lastResponseMetadata, setLastResponseMetadata] = useState<any>(null);
  const [isConsciousnessSubscribed, setIsConsciousnessSubscribed] = useState(false);
  const [conversationId, setConversationId] = useState<string>('');
  const [queryCount, setQueryCount] = useState(0);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    // Only auto-scroll if user has interacted (sent a message)
    // This prevents auto-scroll on initial load
    if (hasUserInteracted) {
      scrollToBottom();
    }
  }, [messages, hasUserInteracted]);

  useEffect(() => {
    // Set up WebSocket connection status monitoring
    setConnectionStatus(cortexWebSocket.getConnectionStatus());
    
    const unsubscribeStatus = cortexWebSocket.onConnectionStatusChange((status) => {
      setConnectionStatus(status);
      
      // Subscribe to consciousness updates and load history when connected
      if (status === 'connected' && !isConsciousnessSubscribed) {
        subscribeToConsciousness();
        loadConversationHistory();
      }
    });

    // Connect if not already connected
    if (cortexWebSocket.getConnectionStatus() === 'disconnected') {
      cortexWebSocket.connect().catch(console.error);
    } else if (cortexWebSocket.getConnectionStatus() === 'connected' && !isConsciousnessSubscribed) {
      subscribeToConsciousness();
      loadConversationHistory();
    }

    // Auto-focus input on mount
    setTimeout(() => {
      inputRef.current?.focus();
    }, 100);

    return () => {
      unsubscribeStatus();
    };
  }, [isConsciousnessSubscribed, historyLoaded]);

  const subscribeToConsciousness = async () => {
    try {
      cortexWebSocket.subscribe('consciousness', (message: any) => {
        setConsciousnessState(message);
        
        // Extract conversation context if available
        if (message?.conversation_context) {
          setConversationId(message.conversation_context.conversation_id || '');
          setQueryCount(message.conversation_context.query_count || 0);
        }
      });
      setIsConsciousnessSubscribed(true);
    } catch (error) {
      console.error('Failed to subscribe to consciousness updates:', error);
    }
  };

  const loadConversationHistory = async () => {
    if (historyLoaded || isLoadingHistory) return;
    
    setIsLoadingHistory(true);
    try {
      console.log('🔄 Loading conversation history...');
      const historyMessages = await cortexWebSocket.getConversationHistory(10);
      
      if (historyMessages && historyMessages.length > 0) {
        console.log(`📚 Loaded ${historyMessages.length} messages from history`);
        setMessages(historyMessages);
      } else {
        console.log('📚 No conversation history found, showing welcome message');
        // Only show welcome message if no history exists
        setMessages([{
          id: '1',
          content: 'Hello! I\'m your Loreum Network assistant. I can help you query the network, search through documents, and answer questions about the blockchain data. I have access to advanced consciousness, memory, and agent systems. How can I assist you today?',
          role: 'assistant',
          timestamp: new Date().toISOString(),
        }]);
      }
      setHistoryLoaded(true);
    } catch (error) {
      console.error('❌ Failed to load conversation history:', error);
      // Show welcome message on error
      setMessages([{
        id: '1',
        content: 'Hello! I\'m your Loreum Network assistant. I can help you query the network, search through documents, and answer questions about the blockchain data. I have access to advanced consciousness, memory, and agent systems. How can I assist you today?',
        role: 'assistant',
        timestamp: new Date().toISOString(),
      }]);
      setHistoryLoaded(true);
    } finally {
      setIsLoadingHistory(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading || connectionStatus !== 'connected') return;

    // Mark that user has interacted (enables auto-scrolling)
    setHasUserInteracted(true);

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      content: input,
      role: 'user',
      timestamp: new Date().toISOString(),
    };

    setMessages(prev => [...prev, userMessage]);
    const queryText = input;
    setInput('');
    setIsLoading(true);

    // Create a placeholder message for streaming
    const assistantMessageId = `response_${Date.now()}`;
    const placeholderMessage: ChatMessage = {
      id: assistantMessageId,
      content: '',
      role: 'assistant',
      timestamp: new Date().toISOString(),
    };

    setMessages(prev => [...prev, placeholderMessage]);
    setIsStreaming(true);

    try {
      const response = await cortexWebSocket.submitQuery(
        queryText, 
        useRAG, 
        (chunk: string, metadata?: any) => {
          // Handle streaming chunks - update the message content
          setMessages(prev => prev.map(msg => 
            msg.id === assistantMessageId 
              ? { ...msg, content: msg.content + chunk }
              : msg
          ));
          
          // Store metadata for debugging/display
          if (metadata) {
            setLastResponseMetadata(metadata);
          }
        }
      );

      // Final update with complete response (in case streaming didn't work)
      setMessages(prev => prev.map(msg => 
        msg.id === assistantMessageId 
          ? { ...msg, content: response || msg.content }
          : msg
      ));

    } catch (error) {
      // Replace the placeholder with error message
      const errorMessage: ChatMessage = {
        id: assistantMessageId,
        content: `Sorry, I encountered an error while processing your request: ${
          error instanceof Error ? error.message : 'Unknown error'
        }`,
        role: 'assistant',
        timestamp: new Date().toISOString(),
      };

      setMessages(prev => prev.map(msg => 
        msg.id === assistantMessageId ? errorMessage : msg
      ));
    } finally {
      setIsLoading(false);
      setIsStreaming(false);
      
      // Auto-refocus the input after response is complete
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  };

  const formatTimestamp = (timestamp: string): string => {
    return new Date(timestamp).toLocaleTimeString([], { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  return (
    <div className="max-w-5xl mx-auto h-screen flex flex-col bg-tesla-black">

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6 bg-tesla-black">
        {isLoadingHistory && (
          <div className="flex gap-4 justify-center">
            <div className="bg-tesla-dark-gray text-tesla-white px-4 py-3 border border-tesla-border">
              <div className="flex items-center gap-3">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="text-sm uppercase tracking-wider">
                  LOADING CONVERSATION HISTORY...
                </span>
              </div>
            </div>
          </div>
        )}
        
        {messages.map((message) => (
          <div
            key={message.id}
            className={`flex gap-4 ${
              message.role === 'user' ? 'justify-end' : 'justify-start'
            }`}
          >
            {message.role === 'assistant' && (
              <div className="flex-shrink-0">
                <div className="w-10 h-10 bg-tesla-white flex items-center justify-center">
                  <Bot className="h-5 w-5 text-tesla-black" />
                </div>
              </div>
            )}
            
            <div
              className={`max-w-xs lg:max-w-lg px-4 py-3 border transition-all duration-150 ${
                message.role === 'user'
                  ? 'bg-tesla-white text-tesla-black border-tesla-white'
                  : 'bg-tesla-dark-gray text-tesla-white border-tesla-border hover:border-tesla-light-gray'
              }`}
            >
              <div 
                className="whitespace-pre-wrap text-sm leading-relaxed prose prose-invert prose-sm max-w-none"
                dangerouslySetInnerHTML={{ 
                  __html: marked(message.content, { 
                    breaks: true,
                    gfm: true 
                  }) 
                }}
              />
              <div className={`text-xs mt-2 uppercase tracking-wider ${
                message.role === 'user' ? 'text-tesla-medium-gray' : 'text-tesla-text-gray'
              }`}>
                {formatTimestamp(message.timestamp)}
                {message.role === 'assistant' && lastResponseMetadata && (
                  <div className="mt-1 text-xs opacity-75">
                    {lastResponseMetadata.source && (
                      <span className="mr-2">
                        SOURCE: {lastResponseMetadata.source.toUpperCase()}
                      </span>
                    )}
                    {lastResponseMetadata.processing_type && (
                      <span className="mr-2">
                        TYPE: {lastResponseMetadata.processing_type.toUpperCase()}
                      </span>
                    )}
                    {lastResponseMetadata.consciousness_cycle && (
                      <span className="mr-2">
                        CYCLE: {lastResponseMetadata.consciousness_cycle}
                      </span>
                    )}
                    {lastResponseMetadata.conversation_id && (
                      <span>
                        CONV: {lastResponseMetadata.conversation_id.slice(-8)}
                      </span>
                    )}
                  </div>
                )}
              </div>
            </div>

            {message.role === 'user' && (
              <div className="flex-shrink-0">
                <div className="w-10 h-10 bg-tesla-medium-gray border border-tesla-border flex items-center justify-center">
                  <User className="h-5 w-5 text-tesla-white" />
                </div>
              </div>
            )}
          </div>
        ))}

        {(isLoading || isStreaming) && (
          <div className="flex gap-4 justify-start">
            <div className="flex-shrink-0">
              <div className="w-10 h-10 bg-tesla-white flex items-center justify-center">
                <Bot className="h-5 w-5 text-tesla-black" />
              </div>
            </div>
            <div className="bg-tesla-dark-gray text-tesla-white px-4 py-3 border border-tesla-border">
              <div className="flex items-center gap-3">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="text-sm uppercase tracking-wider">
                  {isStreaming ? 'STREAMING...' : 'PROCESSING...'}
                </span>
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="bg-tesla-dark-gray border-t border-tesla-border p-6">
        <form onSubmit={handleSubmit} className="flex gap-3">
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="ASK ME ABOUT THE LOREUM NETWORK, BLOCKCHAIN DATA, OR DOCUMENTS..."
            className="flex-1 bg-tesla-medium-gray border border-tesla-border px-4 py-3 text-tesla-white placeholder-tesla-text-gray focus:outline-none focus:border-tesla-white text-sm transition-all duration-150"
            disabled={isLoading}
          />
          <button
            type="submit"
            disabled={!input.trim() || isLoading || connectionStatus !== 'connected'}
            className="tesla-button disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 text-xs"
          >
            <Send className="h-4 w-4" />
            {connectionStatus === 'connected' ? 'SEND' : 'CONNECTING...'}
          </button>
        </form>
      </div>
    </div>
  );
}