# Loreum Cortex

## Tech Stack & Core Components

1. Peer-to-Peer Network with libp2p
- Built in Golang, leveraging libp2p for peer discovery, secure communication, and decentralized networking.
- Nodes operate independently but maintain a shared ledger of reputations and specializations.
- Gossip-based protocol to sync state and optimize query routing.

2. Distributed Retrieval-Augmented Generation (RAG) System
- Every node hosts a local vector-embedded database to process queries efficiently.
- Queries are routed through specialized nodes based on reputation scores and data proximity.
- EigenDA for storage, ensuring scalability and decentralized data access.

3. Modular Sensor & Agent Hub
- Sensors collect real-time and fetch-based data, providing context for decision-making.
- Agents process and act on the data, enabling autonomous workflows across nodes.
- Pluggable design allows developers to add custom sensors and agents.

4. Blockchain as a Global Index
- Layer 2 Rollup to maintain a global registry of nodes and their capabilities.
- Token-based staking ($LORE) to incentivize reliability and accuracy.
- Nodes with high reputation scores get prioritized in query execution.

5. Heterogeneous Inference Network
- Inspired by research on multi-model inference, nodes dynamically choose which LLMs to use based on the query complexity.
- Combines local computation with remote specialized models for efficiency.
- Built-in model distillation to improve response times for repeated queries.

## Hacky & Notable Innovations

🛠 P2P Query Routing with Reputation-Based Load Balancing
- Instead of a single AI model handling all queries, we distribute them based on node expertise.
- Nodes compete for queries but must stake $LORE to participate, reducing spam.
- Dynamic weights adjust reputation scores based on query difficulty and response quality.

🚀 Agent & Sensor Marketplace
- Nodes are extended with AI Agents from other nodes, allowing a marketplace of AI capabilities.
- Nodes are extended with Data Sensors that are consumed by network Agents
- Example: A node specialized in financial analysis can import a DeFi trading agent from another node.

### Why This Tech Stack?
- Golang: Optimized for concurrency, critical for a real-time distributed network.
- libp2p: Decentralized P2P networking with built-in security.
- EigenDA: Off-chain storage that scales with demand.
- Vector DBs: Enables fast contextual search for AI models.
- Layer 2 Blockchain: Provides verifiable state without high costs.

### Key Partner Technologies & Benefits
- EigenLayer: Secure and decentralized data availability for RAG queries.
- libp2p: Handles peer discovery & networking, reducing centralization.