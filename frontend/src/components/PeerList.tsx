import type { Peer } from '../services/api';

interface PeerListProps {
  peers: Peer[];
}

export function PeerList({ peers }: PeerListProps) {
  if (peers.length === 0) {
    return (
      <div className="text-center py-8 text-tesla-text-gray">
        <div className="text-xs uppercase tracking-wider">NO PEERS CONNECTED</div>
      </div>
    );
  }

  return (
    <div className="space-y-2 max-h-80 overflow-y-auto">
      {peers.map((peer, index) => (
        <div
          key={peer.id || index}
          className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150"
        >
          <div className="font-mono text-sm text-tesla-white truncate">
            {peer.id}
          </div>
          {peer.protocols && peer.protocols.length > 0 && (
            <div className="mt-2 text-xs text-tesla-text-gray uppercase tracking-wider">
              PROTOCOLS: {peer.protocols.join(', ')}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}