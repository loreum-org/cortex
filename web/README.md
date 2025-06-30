# Loreum Network Web Interface

A modern React TypeScript application for monitoring and interacting with the Loreum Network.

## Features

### Dashboard
- **Real-time Network Metrics**: Monitor peer connections, data transfer, and network status
- **System Health**: Track uptime, memory usage, CPU cores, and goroutines  
- **Query Processing**: View query success rates, latency, and processing statistics
- **RAG System**: Monitor document count, RAG queries, and system performance
- **Event Log**: Real-time feed of network events and activities
- **Connected Peers**: List of currently connected network peers

### Chat Interface
- **Interactive Query System**: Chat-like interface for querying the Loreum network
- **RAG Integration**: Toggle between general knowledge and document-based responses
- **Real-time Communication**: Direct interaction with the Cortex backend
- **Message History**: Persistent chat history during session

## Tech Stack

- **React 19** with TypeScript
- **Vite** for fast development and building
- **Tailwind CSS** for styling
- **React Router** for navigation
- **Chart.js** with React integration for metrics visualization
- **Lucide React** for icons

## Getting Started

### Prerequisites
- Node.js 20.19.0+ (currently running on 20.18.3 with warnings)
- NPM or Yarn

### Installation

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

### Environment Configuration

The application expects the Cortex backend to be running on `http://localhost:8080`. Update the API base URL in `/src/services/api.ts` if needed.

## API Integration

The web interface connects to the following Cortex API endpoints:

- `/metrics/network` - Network statistics
- `/metrics/system` - System health metrics  
- `/metrics/queries` - Query processing stats
- `/metrics/rag` - RAG system metrics
- `/events` - Event log
- `/node/peers` - Connected peers
- `/query` - Submit queries to the network
- `/health` - Health check

## Development

### Project Structure

```
src/
├── components/           # React components
│   ├── Dashboard.tsx    # Main dashboard view
│   ├── ChatInterface.tsx # Chat interface
│   ├── Layout.tsx       # App layout wrapper
│   ├── MetricCard.tsx   # Reusable metric display
│   ├── EventList.tsx    # Event log component
│   ├── PeerList.tsx     # Peer list component
│   └── MetricChart.tsx  # Chart component
├── services/
│   └── api.ts          # API client and types
├── App.tsx             # Main app component
├── main.tsx            # App entry point
└── index.css           # Global styles
```

### Auto-refresh

The dashboard automatically refreshes every 5 seconds when auto-refresh is enabled. This can be toggled in the dashboard header.

### Error Handling

The application includes comprehensive error handling for API failures and displays user-friendly error messages.

## Styling

The application uses a dark theme optimized for monitoring dashboards:
- Primary color: Purple (#8b5cf6)
- Background: Dark slate colors
- Responsive design with mobile support
- Custom scrollbars and hover effects

## Browser Support

Modern browsers with ES2020+ support required due to Vite configuration.