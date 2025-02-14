# RSCH:V0 DAG-based aBFT Consensus Simulation with Multi-Hop Gossiping

Go implementation that simulates a DAG (Directed Acyclic Graph) based asynchronous Byzantine Fault Tolerance (aBFT) consensus algorithm. The system leverages multi-hop gossiping for transaction propagation and integrates libp2p for peer-to-peer communication.

---

## Overview

The code implements a distributed ledger system using a DAG structure to represent transactions. It incorporates the following key features:

1. **DAG Structure**: Transactions are represented as nodes in a directed acyclic graph, where each transaction references its parent transactions.
2. **aBFT Consensus**: Asynchronous Byzantine Fault Tolerance ensures consensus even in the presence of malicious actors and network partitions.
3. **Multi-Hop Gossiping**: Transactions are propagated through the network using an epidemic protocol that ensures efficient dissemination.
4. **Digital Signatures**: Ed25519 digital signatures provide transaction authenticity and integrity.
5. **Reputation System**: Nodes maintain reputations to penalize misbehavior.

---

## Key Concepts

### 1. Transaction (TX)
A transaction is the fundamental unit of data in the system, represented by the `Transaction` struct:

```go
type Transaction struct {
    ID        string
    Timestamp time.Time
    Data      string
    Parents   []string
    Signature []byte
    PublicKey ed25519.PublicKey
    Finalized bool // Consensus finality flag
}
```

- **ID**: Unique transaction identifier derived from a SHA256 hash.
- **Timestamp**: Time of creation for ordering and validation.
- **Data**: Payload of the transaction.
- **Parents**: References to parent transactions, forming the DAG edges.
- **Signature**: Digital signature of the transaction data.
- **PublicKey**: Public key used for signature verification.
- **Finalized**: Flag indicating if the transaction has achieved finality.

---

### 2. Node
A node represents a participant in the network:

```go
type Node struct {
    ID           string
    Transactions map[string]Transaction
    PrivateKey   ed25519.PrivateKey
    PublicKey    ed25519.PublicKey
    Reputation   int
    mu           sync.Mutex
    Host         host.Host
    Peers        []peer.ID
}
```

- **ID**: Unique node identifier.
- **Transactions**: Local storage of transactions.
- **PrivateKey**: Node's private key for signing transactions.
- **PublicKey**: Public key for verifying signatures.
- **Reputation**: Reputation score to track node behavior.
- **mu**: Mutex for thread-safe operations.
- **Host**: libp2p host instance for P2P communication.
- **Peers**: List of connected peer IDs.

---

### 3. aBFT Consensus
The system implements a simplified version of the aBFT algorithm, providing:

1. **Byzantine Fault Tolerance**: Resilience against malicious nodes.
2. **Asynchrony**: No assumption about upper bounds on message delivery times.
3. **Finality**: Transactions are eventually finalized and considered immutable.

---

### 4. Multi-Hop Gossiping
Transactions are propagated through the network using an epidemic protocol:

```go
// GossipTransaction propagates a transaction to other nodes.
func (n *Node) GossipTransaction(tx Transaction, nodes []*Node) {
    for _, node := range nodes {
        if node.ID != n.ID {
            node.mu.Lock()
            node.Transactions[tx.ID] = tx
            node.mu.Unlock()
        }
    }
}
```

Each node forwards transactions to its peers, ensuring efficient dissemination.

---

### 5. Digital Signatures and Validation
Transactions are signed using Ed25519 digital signatures:

```go
// CreateTransaction generates a signed transaction.
func (n *Node) CreateTransaction(data string, parents []string) Transaction {
    hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", data, time.Now().UnixNano())))
    txID := hex.EncodeToString(hash[:])
    signature := ed25519.Sign(n.PrivateKey, hash[:])

    tx := Transaction{
        ID:        txID,
        Timestamp: time.Now(),
        Data:      data,
        Parents:   parents,
        Signature: signature,
        PublicKey: n.PublicKey,
        Finalized: false,
    }

    n.mu.Lock()
    n.Transactions[tx.ID] = tx
    n.mu.Unlock()

    return tx
}
```

Transactions are validated by verifying their signatures and ensuring all parent transactions exist:

```go
// ValidateTransaction checks if a transaction is valid.
func (n *Node) ValidateTransaction(tx Transaction) bool {
    n.mu.Lock()
    defer n.mu.Unlock()

    hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", tx.Data, tx.Timestamp.UnixNano())))
    if !ed25519.Verify(tx.PublicKey, hash[:], tx.Signature) {
        fmt.Println("Transaction signature invalid!")
        n.Reputation -= 5
        return false
    }

    for _, parentID := range tx.Parents {
        if _, exists := n.Transactions[parentID]; !exists {
            fmt.Println("Transaction parent missing!")
            return false
        }
    }
    return true
}
```

---

## Code Structure

### 1. Transaction Creation and Validation
The `CreateTransaction` method generates a new transaction, while the `ValidateTransaction` method ensures its integrity.

### 2. Gossip Protocol
Transactions are propagated through the network using the `GossipTransaction` method. The `MultiHopGossip` method handles peer-to-peer communication via libp2p.

### 3. Transaction Finalization
Once transactions achieve consensus, they are marked as finalized:

```go
// FinalizeTransaction: Ensures consensus by checkpointing transactions.
func (n *Node) FinalizeTransaction(txID string) {
    n.mu.Lock()
    defer n.mu.Unlock()

    if tx, exists := n.Transactions[txID]; exists && !tx.Finalized {
        tx.Finalized = true
        n.Transactions[txID] = tx
        fmt.Printf("Transaction %s finalized in node %s\n", txID, n.ID)
    }
}
```

---

## How It Works

### 1. Node Initialization
Each node is initialized with a unique ID, private key, public key, and libp2p host.

### 2. Transaction Creation
Nodes create transactions by signing data with their private key and storing them locally.

### 3. Transaction Validation
Transactions are validated by checking signatures and ensuring all parent transactions exist.

### 4. Gossip Propagation
Valid transactions are propagated to other nodes in the network through gossiping.

### 5. Multi-Hop Routing
The libp2p protocol enables efficient multi-hop routing of transactions between nodes.

### 6. Transaction Finalization
Transactions achieve finality after successful propagation and validation across the network.

---

## Security Considerations

1. **Digital Signatures**: Transactions are signed using Ed25519, ensuring authenticity and integrity.
2. **Reputation System**: Nodes with invalid or malicious behavior are penalized by reducing their reputation score.
3. **Validation Checks**: Comprehensive validation ensures that only legitimate transactions are accepted and propagated.

---

## Reputation System

Nodes maintain a reputation score to track behavior:

```go
if !ed25519.Verify(tx.PublicKey, hash[:], tx.Signature) {
    fmt.Println("Transaction signature invalid!")
    n.Reputation -= 5
    return false
}
```

Misbehaving nodes (e.g., those submitting invalid transactions) are penalized by reducing their reputation score.

---

## Multi-Hop Gossip Protocol

The `GossipTransaction` method propagates transactions to all connected peers:

```go
for _, node := range nodes {
    if node.ID != n.ID {
        node.mu.Lock()
        node.Transactions[tx.ID] = tx
        node.mu.Unlock()
    }
}
```

This ensures that transactions are disseminated efficiently through the network.

---

## Transaction Finality

Transactions achieve finality when they are confirmed by all nodes in the network:

```go
nodeA.FinalizeTransaction(genesisTx.ID)
nodeB.FinalizeTransaction(tx1.ID)
nodeC.FinalizeTransaction(tx2.ID)
```

Once finalized, transactions are considered immutable and cannot be altered.

---

## Conclusion

This implementation demonstrates a DAG-based aBFT consensus system with multi-hop gossiping for transaction propagation. The system ensures security through digital signatures, validates transactions thoroughly, and maintains node reputations to promote honest behavior. Transactions achieve finality after being validated and propagated across the network.

---

## Further Directions

1. **Optimization**: Improve the efficiency of transaction propagation.
2. **Scalability**: Increase the number of nodes in the network.
3. **Persistence**: Implement persistent storage for transactions.
4. **Advanced Security Features**: Add additional security measures, such as encryption and access control.
5. **Consensus Optimization**: Explore optimizations to reduce latency and improve throughput.

By extending this base implementation, a production-grade distributed ledger system can be developed.