import { Clock, Info, AlertCircle, CheckCircle, Search, FileText } from 'lucide-react';
import type { EventData } from '../services/api';

interface EventListProps {
  events: EventData[];
}

export function EventList({ events }: EventListProps) {
  const getEventIcon = (type: string) => {
    if (type.includes('error') || type.includes('failure')) {
      return <AlertCircle className="h-4 w-4 text-tesla-accent-red" />;
    }
    if (type.includes('success')) {
      return <CheckCircle className="h-4 w-4 text-tesla-accent-green" />;
    }
    if (type.includes('query')) {
      return <Search className="h-4 w-4 text-tesla-accent-blue" />;
    }
    if (type.includes('document')) {
      return <FileText className="h-4 w-4 text-tesla-white" />;
    }
    return <Info className="h-4 w-4 text-tesla-text-gray" />;
  };

  const formatEventData = (data: Record<string, any>): string => {
    if (!data || Object.keys(data).length === 0) return '';
    
    return Object.entries(data)
      .map(([key, value]) => `${key}: ${value}`)
      .join(' | ');
  };

  if (events.length === 0) {
    return (
      <div className="text-center py-8 text-tesla-text-gray">
        <div className="text-xs uppercase tracking-wider">NO EVENTS RECORDED</div>
      </div>
    );
  }

  return (
    <div className="space-y-2 max-h-80 overflow-y-auto">
      {events.map((event) => (
        <div
          key={event.id}
          className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150"
        >
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              {getEventIcon(event.type)}
              <span className="font-medium text-tesla-white text-sm uppercase tracking-wider">{event.type}</span>
            </div>
            <div className="flex items-center gap-2 text-xs text-tesla-text-gray">
              <Clock className="h-3 w-3" />
              {new Date(event.timestamp).toLocaleTimeString()}
            </div>
          </div>
          {event.data && Object.keys(event.data).length > 0 && (
            <div className="mt-3 text-xs text-tesla-text-gray font-mono">
              {formatEventData(event.data)}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}